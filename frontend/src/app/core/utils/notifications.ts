import type { NotificationItem } from '../models/notifications';

function asRecord(value: unknown): Record<string, unknown> | null {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null;
}

export function formatNotificationContent(content: string | null, data: unknown): string {
  const text = content ?? '';
  const values = asRecord(data);
  if (!values) return text;

  let hasMissingValue = false;
  const formatted = text.replace(/\{([\w.]+)\}/g, (placeholder, key: string) => {
    const value = key.split('.').reduce<unknown>((current, part) => asRecord(current)?.[part], values);
    if (value === undefined || value === null) {
      hasMissingValue = true;
      return placeholder;
    }
    return String(value);
  });
  return hasMissingValue ? text : formatted;
}

export function getExportFileId(item: NotificationItem): string | null {
  const data = asRecord(item.data);
  const status = data?.['status'];
  const fileId = data?.['fileStorageId'];
  return (status === 'Successful' || status === 'Success') && typeof fileId === 'string' && fileId.trim()
    ? fileId : null;
}

export function mergeNotifications(
  incoming: readonly NotificationItem[],
  current: readonly NotificationItem[],
): NotificationItem[] {
  const byId = new Map(incoming.map((item) => [item.notificationId, item]));
  for (const item of current) {
    const other = byId.get(item.notificationId);
    if (!other || Date.parse(item.sentAt) >= Date.parse(other.sentAt)) {
      byId.set(item.notificationId, item);
    }
  }
  return [...byId.values()].sort((left, right) => Date.parse(right.sentAt) - Date.parse(left.sentAt));
}
