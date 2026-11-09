import { environment } from '../../../environments/environment';
import { ApiSource } from '../enums/api-sources';

export const CLIENT_ACTION_ID_HEADER = 'X-Client-Action-Id';
export const CLIENT_REQUEST_ID_HEADER = 'X-Client-Request-Id';
export const API_BASE_URLS_MAPPING: Record<ApiSource, string> = {
  [ApiSource.Organization]: environment.api.baseUrl,
  [ApiSource.Auth]: environment.api.authUrl,
  [ApiSource.Notification]: environment.api.notificationUrl,
};

export const CSRF_COOKIE_NAME = 'XSRF-TOKEN';
export const CSRF_HEADER_NAME = 'X-XSRF-TOKEN';
export const CSRF_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);
export const API_BASE_URLS = Object.values(environment.api).map((url) => url.replace(/\/+$/, ''));
