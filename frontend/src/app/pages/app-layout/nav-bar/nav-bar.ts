import { CdkOverlayOrigin } from '@angular/cdk/overlay';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  input,
  OnInit,
  output,
  signal,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { TpAvatar, TpBellNotification, TpTextButton, TpSearchBar, TpButton } from '@team-platform/ui';
import { finalize } from 'rxjs';
import { BellNotificationItem, NotificationItem } from '../../../core/models/notifications';
import { ClientCorrelationService } from '../../../core/services/client-correlation-service';
import { NotificationService } from '../../../core/services/notification-service';
import { mergeNotifications } from '../../../core/utils/notifications';
import { BellNotificationMenu } from './bell-notification-menu/bell-notification-menu';
import { UserAccountMenu } from './user-account-menu/user-account-menu';

@Component({
  selector: 'app-nav-bar',
  imports: [
    CdkOverlayOrigin,
    TpAvatar,
    TpBellNotification,
    TpTextButton,
    BellNotificationMenu,
    UserAccountMenu,
    TpSearchBar,
    TpButton
],
  templateUrl: './nav-bar.html',
  styleUrl: './nav-bar.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class NavBar implements OnInit {
  private readonly destroyRef = inject(DestroyRef);
  private readonly clientCorrelationService = inject(ClientCorrelationService);
  private readonly notificationService = inject(NotificationService);

  readonly sideBarCollapsed = input(false);
  readonly sideBarToggle = output<void>();

  readonly isBellNotificationMenuOpen = signal(false);
  readonly isUserAccountMenuOpen = signal(false);
  readonly isUnreadOnly = signal(false);
  readonly isLoadingNotifications = signal(false);
  readonly notifications = signal<NotificationItem[]>([]);
  readonly visibleNotifications = computed(() =>
    this.isUnreadOnly()
      ? this.notifications().filter((notification) => notification.isRead !== true)
      : this.notifications(),
  );
  readonly unreadCount = computed(
    () => this.notifications().filter((notification) => notification.isRead !== true).length,
  );
  readonly isMarkingAllNotificationsRead = signal(false);

  private readonly pendingMarkReadIds = new Set<string>();
  private readonly inFlightMarkReadIds = new Set<string>();
  private activeNotificationRequests = 0;
  private pendingMarkReadTimer: ReturnType<typeof setTimeout> | null = null;
  private successfulMarkAllReadCount = 0;
  private notificationRevision = 0;
  private readonly realtimeRevisions = new Map<string, number>();

  ngOnInit(): void {
    this.notificationService.bellNotifications$
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((notification) => this.addNotification(notification));

    const hubActionId = this.clientCorrelationService.startAction();

    void this.notificationService
      .startHubConnection()
      .catch((error: unknown) => {
        console.error('Failed to connect to the notification hub.', error);
      })
      .finally(() => {
        if (this.clientCorrelationService.getClientActionId() === hubActionId) {
          this.clientCorrelationService.completeAction();
        }

        if (this.destroyRef.destroyed) {
          return;
        }

        this.loadNotifications();
      });

    this.destroyRef.onDestroy(() => {
      this.clearPendingMarkReadTimer();
      this.pendingMarkReadIds.clear();
      void this.notificationService.stopHubConnection().catch((error: unknown) => {
        console.error('Failed to stop the notification hub connection.', error);
      });
    });
  }

  scheduleMarkNotificationRead(notificationId: string): void {
    if (
      this.isMarkingAllNotificationsRead() ||
      this.pendingMarkReadIds.has(notificationId) ||
      this.inFlightMarkReadIds.has(notificationId)
    ) {
      return;
    }

    const notification = this.notifications().find(
      (item) => item.notificationId === notificationId,
    );
    if (!notification || notification.isRead === true) return;

    const readAt = new Date().toISOString();
    this.notifications.update((notifications) =>
      notifications.map((item) =>
        item.notificationId === notificationId ? { ...item, isRead: true, readAt } : item,
      ),
    );
    this.pendingMarkReadIds.add(notificationId);
    this.schedulePendingMarkRead();
  }

  markAllNotificationsRead(): void {
    if (
      this.isMarkingAllNotificationsRead() ||
      !this.notifications().some((notification) => notification.isRead !== true)
    ) {
      return;
    }

    this.clearPendingMarkReadTimer();
    this.isMarkingAllNotificationsRead.set(true);
    const requestRevision = this.notificationRevision;

    this.clientCorrelationService.startAction();
    this.notificationService
      .markNotifications({ isAll: true })
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => {
          this.clientCorrelationService.completeAction();
          this.isMarkingAllNotificationsRead.set(false);
          if (!this.destroyRef.destroyed && this.pendingMarkReadIds.size > 0) {
            this.schedulePendingMarkRead();
          }
        }),
      )
      .subscribe({
        next: () => {
          this.successfulMarkAllReadCount++;
          for (const id of this.pendingMarkReadIds) {
            if ((this.realtimeRevisions.get(id) ?? 0) <= requestRevision) this.pendingMarkReadIds.delete(id);
          }
          const readAt = new Date().toISOString();
          this.notifications.update((notifications) =>
            notifications.map((notification) =>
              notification.isRead === true || (this.realtimeRevisions.get(notification.notificationId) ?? 0) > requestRevision
                ? notification
                : { ...notification, isRead: true, readAt },
            ),
          );
        },
        error: (error: unknown) => console.error('Failed to mark all notifications read.', error),
      });
  }

  toggleBellNotificationMenu(): void {
    this.isBellNotificationMenuOpen.update((isOpen) => !isOpen);
  }

  toggleUserAccountMenu(): void {
    this.isUserAccountMenuOpen.update((isOpen) => !isOpen);
  }

  private addNotification(notification: BellNotificationItem): void {
    const current = this.notifications().find((item) => item.notificationId === notification.notificationId);
    if (current && Date.parse(notification.sentAt) < Date.parse(current.sentAt)) return;
    if (current && notification.sentAt === current.sentAt && current.isRead && !notification.isRead) return;
    this.realtimeRevisions.set(notification.notificationId, ++this.notificationRevision);
    this.pendingMarkReadIds.delete(notification.notificationId);
    this.notifications.update((notifications) => {
      return [
        notification,
        ...notifications.filter((item) => item.notificationId !== notification.notificationId),
      ];
    });
  }

  loadNotifications(isUnreadOnly = this.isUnreadOnly()): void {
    const requestRevision = this.notificationRevision;
    const requestActionId = this.clientCorrelationService.startAction();
    this.activeNotificationRequests++;
    this.isLoadingNotifications.set(true);

    this.notificationService
      .getLatestNotificationForUser(isUnreadOnly)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => {
          this.activeNotificationRequests--;
          this.isLoadingNotifications.set(this.activeNotificationRequests > 0);
          if (this.clientCorrelationService.getClientActionId() === requestActionId) {
            this.clientCorrelationService.completeAction();
          }
        }),
      )
      .subscribe({
        next: (notifications) => {
          if (isUnreadOnly !== this.isUnreadOnly()) return;

          if (isUnreadOnly) {
            const recent = this.notifications().filter(
              (item) => (this.realtimeRevisions.get(item.notificationId) ?? 0) > requestRevision,
            );
            this.notifications.set(mergeNotifications(notifications, recent));
          } else {
            this.mergeNotifications(notifications);
          }
        },
        error: (error: unknown) => console.error('Failed to load notifications.', error),
      });
  }

  private markPendingNotificationsRead(): void {
    if (this.destroyRef.destroyed || this.isMarkingAllNotificationsRead()) return;

    const notificationIds = [...this.pendingMarkReadIds];
    if (notificationIds.length === 0) return;

    this.pendingMarkReadIds.clear();
    notificationIds.forEach((notificationId) => this.inFlightMarkReadIds.add(notificationId));
    const notificationIdSet = new Set(notificationIds);
    const requestRevision = this.notificationRevision;
    const markAllReadCount = this.successfulMarkAllReadCount;

    const actionId = this.clientCorrelationService.startAction();
    this.notificationService
      .markNotifications({ notificationIds, isAll: false })
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => {
          notificationIds.forEach((notificationId) =>
            this.inFlightMarkReadIds.delete(notificationId),
          );
          if (this.clientCorrelationService.getClientActionId() === actionId) {
            this.clientCorrelationService.completeAction();
          }
        }),
      )
      .subscribe({
        error: (error: unknown) => {
          if (this.successfulMarkAllReadCount === markAllReadCount) {
            this.notifications.update((notifications) =>
              notifications.map((notification) =>
                notificationIdSet.has(notification.notificationId)
                  && (this.realtimeRevisions.get(notification.notificationId) ?? 0) <= requestRevision
                  ? { ...notification, isRead: false, readAt: null }
                  : notification,
              ),
            );
          }
          console.error('Failed to mark notification read.', error);
        },
      });
  }

  private schedulePendingMarkRead(): void {
    this.clearPendingMarkReadTimer();
    this.pendingMarkReadTimer = setTimeout(() => {
      this.pendingMarkReadTimer = null;
      this.markPendingNotificationsRead();
    }, 500);
  }

  private clearPendingMarkReadTimer(): void {
    if (this.pendingMarkReadTimer !== null) {
      clearTimeout(this.pendingMarkReadTimer);
      this.pendingMarkReadTimer = null;
    }
  }

  private mergeNotifications(incoming: NotificationItem[]): void {
    this.notifications.update((current) => mergeNotifications(incoming, current));
  }
}
