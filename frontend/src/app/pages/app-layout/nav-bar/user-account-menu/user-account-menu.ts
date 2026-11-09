import { CdkOverlayOrigin } from '@angular/cdk/overlay';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  model,
} from '@angular/core';
import { Router } from '@angular/router';
import { TpDivider, TpPopoverMenu, TpTextButton } from '@team-platform/ui';
import { LogoutRequest } from '../../../../core/models/auth';
import { AuthService } from '../../../../core/services/auth-service';
import { ClientCorrelationService } from '../../../../core/services/client-correlation-service';

@Component({
  selector: 'app-user-account-menu',
  imports: [TpPopoverMenu, TpDivider, TpTextButton],
  templateUrl: './user-account-menu.html',
  styleUrl: './user-account-menu.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UserAccountMenu {
  private readonly authService = inject(AuthService);
  private readonly clientCorrelationService = inject(ClientCorrelationService);
  private readonly router = inject(Router);

  readonly open = model(false);
  readonly origin = input<CdkOverlayOrigin | null>(null);
  readonly currentUser = computed(() => this.authService.currentUser());

  protected navigateTo(path: string): void {
    this.open.set(false);
    this.router.navigateByUrl(path);
  }

  logout(): void {
    this.open.set(false);
    const returnUrl = window.location.origin;
    const request: LogoutRequest = { returnUrl };

    this.clientCorrelationService.startAction();
    this.authService.logout(request).subscribe({
      next: (res) => {
        window.location.assign(res.logoutUrl);
      },
    });
    this.clientCorrelationService.completeAction();
  }
}
