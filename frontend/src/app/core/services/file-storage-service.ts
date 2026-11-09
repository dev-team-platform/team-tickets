import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import type { Observable } from 'rxjs';
import { AppHttpContext } from './http-context';

@Injectable({ providedIn: 'root' })
export class FileStorageService {
  private readonly http = inject(HttpClient);
  private readonly context = inject(AppHttpContext);

  getDownloadUrl(fileId: string): Observable<{ url: string }> {
    return this.http.post<{ url: string }>('file-storages/download', { fileId }, {
      context: this.context.baseContext,
    });
  }
}
