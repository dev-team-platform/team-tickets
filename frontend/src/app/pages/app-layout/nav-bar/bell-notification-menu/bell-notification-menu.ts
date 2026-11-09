import { CdkOverlayOrigin } from '@angular/cdk/overlay';
import { DatePipe, NgTemplateOutlet } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  TemplateRef,
  computed,
  input,
  model,
  output,
  viewChild,
  inject,
} from '@angular/core';
import { finalize } from 'rxjs';
import { RouterLink } from '@angular/router';
import {
  TpDivider,
  TpPopoverMenu,
  TpSkeleton,
  TpSlideToggle,
  TpTextButton,
} from '@team-platform/ui';
import { BELL_NOTIFICATION_TITLE } from '../../../../core/constants/notifications';
import { NotificationItem } from '../../../../core/models/notifications';
import { ClientCorrelationService } from '../../../../core/services/client-correlation-service';
import { FileStorageService } from '../../../../core/services/file-storage-service';
import { formatNotificationContent, getExportFileId } from '../../../../core/utils/notifications';
import { ToastService } from '../../../../core/services/toast-service';
import { resolveErrorMessage } from '../../../../core/utils/common';

interface NotificationContentContext {
  $implicit: NotificationItem;
}

type NotificationTitle = (typeof BELL_NOTIFICATION_TITLE)[keyof typeof BELL_NOTIFICATION_TITLE];

@Component({
  selector: 'app-bell-notification-menu',
  imports: [
    DatePipe,
    NgTemplateOutlet,
    RouterLink,
    TpDivider,
    TpPopoverMenu,
    TpSkeleton,
    TpSlideToggle,
    TpTextButton,
  ],
  templateUrl: './bell-notification-menu.html',
  styleUrl: './bell-notification-menu.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BellNotificationMenu {
  private readonly fileStorageService = inject(FileStorageService);
  private readonly correlation = inject(ClientCorrelationService);
  private readonly toast = inject(ToastService);
  readonly open = model(false);
  readonly isUnreadOnly = model(false);
  readonly origin = input<CdkOverlayOrigin | null>(null);
  readonly notifications = input<NotificationItem[]>([]);
  readonly loading = input(false);
  readonly markAllReadDisabled = input(false);
  readonly markAsRead = output<string>();
  readonly markAllRead = output<void>();
  protected readonly canMarkAllRead = computed(
    () =>
      !this.loading() &&
      !this.markAllReadDisabled() &&
      this.notifications().some((notification) => notification.isRead !== true),
  );
  protected readonly skeletonItems = [0, 1, 2, 3, 4];
  private readonly defaultContentTemplate =
    viewChild.required<TemplateRef<NotificationContentContext>>('defaultContentTemplate');

  // Register templates here only for notification types that need interactive content.
  // Plain placeholder replacement is handled by the default template.
  private readonly interactiveContentTemplates: Partial<
    Record<NotificationTitle, () => TemplateRef<NotificationContentContext>>
  > = {};

  protected closeMenu(): void {
    this.open.set(false);
  }

  protected markNotificationRead(notification: NotificationItem): void {
    if (notification.isRead === true) return;

    this.markAsRead.emit(notification.notificationId);
  }

  protected handleNotificationKeydown(event: KeyboardEvent, notification: NotificationItem): void {
    if (event.key !== 'Enter' && event.key !== ' ') return;

    event.preventDefault();
    this.markNotificationRead(notification);
  }

  formatContent(item: NotificationItem): TemplateRef<NotificationContentContext> {
    const templateResolver = Object.entries(this.interactiveContentTemplates).find(([title]) =>
      item.title?.includes(title),
    )?.[1];

    return templateResolver?.() ?? this.defaultContentTemplate();
  }

  protected interpolateContentTemplate(item: NotificationItem): string {
    return formatNotificationContent(item.content, item.data);
  }

  protected exportFileId(item: NotificationItem): string | null {
    return getExportFileId(item);
  }

  protected downloadExport(event: MouseEvent, item: NotificationItem): void {
    event.preventDefault();
    event.stopPropagation();
    const anchor = event.currentTarget as HTMLAnchorElement | null;
    const fileId = this.exportFileId(item);
    if (!fileId) return;
    const actionId = this.correlation.startAction();
    this.fileStorageService.getDownloadUrl(fileId)
      .pipe(finalize(() => {
        if (this.correlation.getClientActionId() === actionId) this.correlation.completeAction();
      }))
      .subscribe({
        next: ({ url }) => {
          if (anchor) anchor.href = url;
          window.location.assign(url);
        },
        error: (error: unknown) => this.toast.enqueue({ type: 'error', content: resolveErrorMessage(error) }),
      });
  }
}
