import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  signal,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { TpSkeleton } from '@team-platform/ui';
import { filter } from 'rxjs';
import { MainSkeleton } from '../../components/main-skeleton/main-skeleton';
import { AuthService } from '../../core/services/auth-service';
import { Breadcrumb } from './breadcrumb/breadcrumb';
import { NavBar } from './nav-bar/nav-bar';
import { SideBar } from './side-bar/side-bar';

@Component({
  selector: 'app-app-layout',
  imports: [NavBar, SideBar, Breadcrumb, RouterOutlet, MainSkeleton, TpSkeleton],
  templateUrl: './app-layout.html',
  styleUrl: './app-layout.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppLayout {
  private readonly authService = inject(AuthService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly router = inject(Router);

  protected readonly latestSearch = signal('');
  protected readonly sideBarCollapsed = signal(false);
  protected readonly currentUrl = signal(this.router.url);
  protected readonly sideBarSkeletonItems = [
    { width: '58%', child: false },
    { width: '74%', child: false },
    { width: '64%', child: true },
    { width: '48%', child: false },
  ];
  protected readonly showInitialSkeleton = computed(
    () => this.authService.currentUser() === null && !this.currentUrl().startsWith('/auth/login'),
  );

  constructor() {
    this.router.events
      .pipe(
        filter((event): event is NavigationEnd => event instanceof NavigationEnd),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe((event) => this.currentUrl.set(event.urlAfterRedirects));
  }

  protected handleSearch(query: string): void {
    this.latestSearch.set(query);
  }

  protected toggleSideBar(): void {
    this.sideBarCollapsed.update((collapsed) => !collapsed);
  }

  protected expandSideBar(): void {
    this.sideBarCollapsed.set(false);
  }
}
