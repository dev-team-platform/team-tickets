import { HttpInterceptorFn } from '@angular/common/http';
import {
  API_BASE_URLS,
  CSRF_COOKIE_NAME,
  CSRF_HEADER_NAME,
  CSRF_METHODS,
} from '../constants/interceptors';
import { getCookie } from '../utils/cookie';

export const csrfInterceptor: HttpInterceptorFn = (req, next) => {
  // 1. Check internal API
  const isApiRequest = API_BASE_URLS.some(
    (baseUrl) => req.url === baseUrl || req.url.startsWith(`${baseUrl}/`),
  );

  // 2. Check HTTP method
  const requiresCsrf = CSRF_METHODS.has(req.method.toUpperCase());

  if (!isApiRequest || !requiresCsrf) {
    return next(req);
  }

  // 3. Get CSRF token
  const csrfToken = getCookie(CSRF_COOKIE_NAME);

  if (!csrfToken) {
    return next(req);
  }

  // 4. Attach CSRF header
  return next(
    req.clone({
      setHeaders: {
        [CSRF_HEADER_NAME]: csrfToken,
      },
      withCredentials: true,
    }),
  );
};
