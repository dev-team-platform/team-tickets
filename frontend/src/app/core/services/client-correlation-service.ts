import { Injectable } from '@angular/core';

export const CORRELATION_KEYS = {
  CLIENT_ACTION_ID: 'TEAM_TICKET_CLIENT_ACTION_ID',
} as const;

@Injectable({
  providedIn: 'root',
})
export class ClientCorrelationService {
  getClientActionId(): string | null {
    return sessionStorage.getItem(CORRELATION_KEYS.CLIENT_ACTION_ID);
  }

  startAction(): string {
    const clientActionId = crypto.randomUUID();

    sessionStorage.setItem(CORRELATION_KEYS.CLIENT_ACTION_ID, clientActionId);

    return clientActionId;
  }

  getOrStartAction(): string {
    return this.getClientActionId() ?? this.startAction();
  }

  completeAction(): void {
    sessionStorage.removeItem(CORRELATION_KEYS.CLIENT_ACTION_ID);
  }

  createRequestId(): string {
    return crypto.randomUUID();
  }
}
