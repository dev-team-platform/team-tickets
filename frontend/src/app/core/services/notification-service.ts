import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import {
  HttpTransportType,
  HubConnection,
  HubConnectionBuilder,
  HubConnectionState,
} from '@microsoft/signalr';
import { delay, Observable, Subject } from 'rxjs';
import { environment } from '../../../environments/environment';
import {
  CLIENT_ACTION_ID_HEADER,
  CLIENT_REQUEST_ID_HEADER,
  CSRF_COOKIE_NAME,
  CSRF_HEADER_NAME,
} from '../constants/interceptors';
import { SOURCE_APP } from '../constants/source-app';
import {
  BellNotificationItem,
  GetAllNotificationsRequest,
  GetAllNotificationsResponse,
  GetLatestNotificationForUserResponse,
  MarkNotificationsRequest,
  MarkNotificationsResponse,
} from '../models/notifications';
import { getCookie } from '../utils/cookie';
import { ClientCorrelationService } from './client-correlation-service';
import { AppHttpContext } from './http-context';

@Injectable({
  providedIn: 'root',
})
export class NotificationService {
  private readonly httpClient = inject(HttpClient);
  private readonly appHttpContext = inject(AppHttpContext);
  private readonly clientCorrelationService = inject(ClientCorrelationService);

  private readonly bellNotificationSubject = new Subject<BellNotificationItem>();

  private hubConnection: HubConnection | null = null;
  private hubConnectionStart: Promise<void> | null = null;

  readonly bellNotifications$: Observable<BellNotificationItem> =
    this.bellNotificationSubject.asObservable();

  getLatestNotificationForUser(
    isUnreadOnly = false,
  ): Observable<GetLatestNotificationForUserResponse> {
    return this.httpClient
      .post<GetLatestNotificationForUserResponse>(
        'notifications/latest',
        { sourceApp: SOURCE_APP, isUnreadOnly },
        { context: this.appHttpContext.notificationContext },
      )
      .pipe(delay(0));
  }

  getAllNotifications(
    request: GetAllNotificationsRequest,
  ): Observable<GetAllNotificationsResponse> {
    return this.httpClient.post<GetAllNotificationsResponse>(
      'notifications/all',
      { ...request, sourceApp: SOURCE_APP },
      { context: this.appHttpContext.notificationContext },
    );
  }

  markNotifications(request: MarkNotificationsRequest): Observable<MarkNotificationsResponse> {
    return this.httpClient.patch<MarkNotificationsResponse>(
      'notifications/mark-as-read',
      { ...request, sourceApp: SOURCE_APP },
      { context: this.appHttpContext.notificationContext },
    );
  }

  startHubConnection(): Promise<void> {
    const connection = this.getHubConnection();

    if (connection.state !== HubConnectionState.Disconnected) {
      return this.hubConnectionStart ?? Promise.resolve();
    }

    const startPromise = connection.start().finally(() => {
      if (this.hubConnectionStart === startPromise) {
        this.hubConnectionStart = null;
      }
    });

    this.hubConnectionStart = startPromise;
    return startPromise;
  }

  async stopHubConnection(): Promise<void> {
    if (this.hubConnection) {
      await this.hubConnection.stop();
    }
  }

  private getHubConnection(): HubConnection {
    if (!this.hubConnection) {
      const hubUrl = `${environment.api.notificationUrl.replace(/\/+$/, '')}/hubs/notifications?source_app=${encodeURIComponent(SOURCE_APP)}`;
      const headers: Record<string, string> = {
        [CLIENT_ACTION_ID_HEADER]: this.clientCorrelationService.getOrStartAction(),
        [CLIENT_REQUEST_ID_HEADER]: this.clientCorrelationService.createRequestId(),
      };
      const csrfToken = getCookie(CSRF_COOKIE_NAME);

      if (csrfToken) {
        headers[CSRF_HEADER_NAME] = csrfToken;
      }

      this.hubConnection = new HubConnectionBuilder()
        .withUrl(hubUrl, {
          headers,
          withCredentials: true,
          transport: HttpTransportType.WebSockets,
        })
        .withAutomaticReconnect()
        .build();

      this.hubConnection.on('BellNotification', (notification: BellNotificationItem) => {
        this.bellNotificationSubject.next(notification);
      });
    }

    return this.hubConnection;
  }
}
