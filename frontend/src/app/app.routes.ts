import { Routes } from '@angular/router';

import { authGuard } from './core/guards/auth-guard';
import { guestGuard } from './core/guards/guest-guard';

import { Login } from './pages/login/login';
import { NotFound } from './pages/not-found/not-found';

export const routes: Routes = [
  // Public routes
  {
    path: 'auth/login',
    component: Login,
    canActivate: [guestGuard],
  },

  // Protected routes
  {
    path: '',
    canActivate: [authGuard],
    canActivateChild: [authGuard],
    children: [
      {
        path: '',
        pathMatch: 'full',
        redirectTo: 'home',
      },
      {
        path: 'home',
        data: {
          breadcrumb: 'Home',
        },
        loadComponent: () => import('./pages/home/home').then((m) => m.Home),
      },
      {
        path: '**',
        component: NotFound,
      },
    ],
  },
];
