import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideRouter } from '@angular/router';

import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { routes } from './app.routes';
import { apiBaseUrlInterceptor } from './core/interceptors/api-base-url-interceptor';
import { csrfInterceptor } from './core/interceptors/csrf-interceptor';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes),
    provideHttpClient(withInterceptors([apiBaseUrlInterceptor, csrfInterceptor])),
  ],
};
