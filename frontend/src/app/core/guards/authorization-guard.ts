import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { Permission, RoleName } from '../enums/uam';
import { AuthService } from '../services/auth-service';

export const authorizationGuard: CanActivateFn = (route) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  const currentUser = authService.currentUser();

  if (!currentUser) {
    return router.createUrlTree(['/page-not-found']);
  }

  const requiredRoles = (route.data['roles'] as RoleName[] | undefined) ?? [];

  const requiredPermissions = (route.data['permissions'] as Permission[] | undefined) ?? [];

  const hasRole = requiredRoles.length === 0 || requiredRoles.includes(currentUser.roleName);

  const hasPermissions =
    requiredPermissions.length === 0 ||
    requiredPermissions.every((permission) => currentUser.permissions.includes(permission));

  return hasRole && hasPermissions ? true : router.createUrlTree(['/page-not-found']);
};
