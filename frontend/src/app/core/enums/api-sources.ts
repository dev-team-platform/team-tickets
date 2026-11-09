import { HttpContextToken } from '@angular/common/http';

export enum ApiSource {
  Organization = 'Organization',
  Auth = 'Auth',
  Notification = 'Notification',
}

export const API_SOURCE = new HttpContextToken<ApiSource>(() => ApiSource.Organization);
