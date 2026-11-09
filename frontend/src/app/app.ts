import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TpToastNotificationContainer } from '@team-platform/ui';
import { ToastService } from './core/services/toast-service';
import { AppLayout } from './pages/app-layout/app-layout';

@Component({
  selector: 'app-root',
  imports: [TpToastNotificationContainer, AppLayout],
  templateUrl: './app.html',
  styleUrl: './app.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class App {
  readonly toastService = inject(ToastService);
}
