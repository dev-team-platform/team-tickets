import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRouteSnapshot, NavigationEnd, Router, RouterLink } from '@angular/router';
import { filter } from 'rxjs';

interface BreadcrumbItem {
  label: string;
  url: string | null;
}

@Component({
  selector: 'app-breadcrumb',
  imports: [RouterLink],
  templateUrl: './breadcrumb.html',
  styleUrl: './breadcrumb.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Breadcrumb {
  protected readonly breadcrumbs = signal<BreadcrumbItem[]>([]);

  private readonly destroyRef = inject(DestroyRef);
  private readonly router = inject(Router);

  constructor() {
    this.updateBreadcrumbs();

    this.router.events
      .pipe(
        filter((event): event is NavigationEnd => event instanceof NavigationEnd),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe(() => this.updateBreadcrumbs());
  }

  private updateBreadcrumbs(): void {
    const breadcrumbs: BreadcrumbItem[] = [];
    let route = this.router.routerState.snapshot.root;

    while (route.firstChild) {
      route = route.firstChild;
      const label = route.data['breadcrumb'];

      if (typeof label === 'string' && label.trim().length > 0) {
        breadcrumbs.push({
          label,
          url: this.getBreadcrumbUrl(route),
        });
      }
    }

    this.breadcrumbs.set(breadcrumbs);
  }

  private getBreadcrumbUrl(route: ActivatedRouteSnapshot): string | null {
    const configuredUrl = route.data['breadcrumbUrl'];

    if (typeof configuredUrl === 'string') {
      return configuredUrl;
    }

    const routeHasPage =
      route.routeConfig?.component !== undefined || route.routeConfig?.loadComponent !== undefined;

    if (!routeHasPage) {
      return null;
    }

    const segments = route.pathFromRoot.flatMap((snapshot) => snapshot.url.map((segment) => segment.path));
    return `/${segments.join('/')}`;
  }
}
