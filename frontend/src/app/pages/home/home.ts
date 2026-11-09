import { JsonPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject, OnInit } from '@angular/core';
import { AuthService } from '../../core/services/auth-service';
import { ClientCorrelationService } from '../../core/services/client-correlation-service';

@Component({
  selector: 'app-home',
  imports: [JsonPipe],
  templateUrl: './home.html',
  styleUrl: './home.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Home implements OnInit {
  private readonly authService = inject(AuthService);
  private readonly clientCorrelationService = inject(ClientCorrelationService);
  readonly currentUser = this.authService.currentUser;

  ngOnInit(): void {
    this.clientCorrelationService.completeAction();
  }
}
