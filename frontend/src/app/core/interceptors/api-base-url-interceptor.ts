import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import {
  API_BASE_URLS_MAPPING,
  CLIENT_ACTION_ID_HEADER,
  CLIENT_REQUEST_ID_HEADER,
} from '../constants/interceptors';
import { API_SOURCE } from '../enums/api-sources';
import { ClientCorrelationService } from '../services/client-correlation-service';

export const apiBaseUrlInterceptor: HttpInterceptorFn = (req, next) => {
  const correlationService = inject(ClientCorrelationService);

  // 1. Resolve request URL
  let requestUrl = req.url;

  if (!/^https?:\/\//i.test(requestUrl)) {
    const source = req.context.get(API_SOURCE);
    const baseUrl = API_BASE_URLS_MAPPING[source].replace(/\/+$/, '');
    const endpoint = requestUrl.replace(/^\/+/, '');

    requestUrl = `${baseUrl}/${endpoint}`;
  }

  // 2. Correlation headers
  const clientActionId = correlationService.getOrStartAction();
  const clientRequestId = correlationService.createRequestId();

  const headers = req.headers
    .set(CLIENT_ACTION_ID_HEADER, clientActionId)
    .set(CLIENT_REQUEST_ID_HEADER, clientRequestId);

  // 3. Forward request
  return next(
    req.clone({
      url: requestUrl,
      headers,
      withCredentials: true,
    }),
  );
};
