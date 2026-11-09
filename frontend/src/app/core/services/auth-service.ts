import { HttpClient } from '@angular/common/http';
import { inject, Injectable, signal } from '@angular/core';
import { catchError, map, Observable, of, tap } from 'rxjs';
import { CurrentUser, LogoutRequest, LogoutResponse } from '../models/auth';
import { AppHttpContext } from './http-context';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly httpClient = inject(HttpClient);
  private readonly appHttpContext = inject(AppHttpContext);

  private readonly _currentUser = signal<CurrentUser | null>(null);

  readonly currentUser = this._currentUser.asReadonly();

  checkAuthentication(): Observable<boolean> {
    return this.httpClient
      .get<CurrentUser>('users/me', {
        context: this.appHttpContext.baseContext,
        withCredentials: true,
      })
      .pipe(
        tap((user) => {
          this._currentUser.set(user);
        }),
        map(() => true),
        catchError(() => {
          this._currentUser.set(null);
          return of(false);
        }),
      );
  }

  initializeCsrf(): Observable<void> {
    return this.httpClient.get<void>('csrf', {
      context: this.appHttpContext.authContext,
      withCredentials: true,
    });
  }

  login(returnUrl: string = '/'): void {
    window.location.href = '/api/v1/auth/login' + '?returnUrl=' + encodeURIComponent(returnUrl);
  }

  logout(req: LogoutRequest): Observable<LogoutResponse> {
    return this.httpClient.post<LogoutResponse>('logout', req, {
      context: this.appHttpContext.authContext,
      withCredentials: true,
    });
  }
}
