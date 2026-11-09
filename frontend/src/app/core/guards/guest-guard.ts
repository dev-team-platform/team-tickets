import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { map } from 'rxjs';
import { AuthService } from '../services/auth-service';

export const guestGuard: CanActivateFn = (route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  return authService.checkAuthentication().pipe(
    map((isAuthenticated) => {
      if (!isAuthenticated) {
        return true;
      }

      const returnUrl = route.queryParamMap.get('returnUrl') ?? '/';

      return router.parseUrl(returnUrl);
    }),
  );
};
