import { Injectable, signal } from '@angular/core';
import { TpToastNotificationItem } from '@team-platform/ui';

@Injectable({
  providedIn: 'root',
})
export class ToastService {
  readonly toasts = signal<TpToastNotificationItem[]>([]);
  private nextId = 0;

  enqueue(item: Omit<TpToastNotificationItem, 'id'>): void {
    this.toasts.update((items) => [...items, { ...item, id: ++this.nextId }]);
  }
}
