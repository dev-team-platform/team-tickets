import { HttpContext } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { API_SOURCE, ApiSource } from '../enums/api-sources';

@Injectable({
  providedIn: 'root',
})
export class AppHttpContext {
  readonly baseContext = new HttpContext().set(API_SOURCE, ApiSource.Organization);
  readonly authContext = new HttpContext().set(API_SOURCE, ApiSource.Auth);
  readonly notificationContext = new HttpContext().set(API_SOURCE, ApiSource.Notification);
}
