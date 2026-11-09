import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { AppHttpContext } from './http-context';

@Injectable({
  providedIn: 'root',
})
export class UserService {
  private readonly httpClient = inject(HttpClient);
  private readonly appHttpContext = inject(AppHttpContext);
}
