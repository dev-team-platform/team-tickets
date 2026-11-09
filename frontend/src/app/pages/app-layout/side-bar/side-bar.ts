import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  output,
  Signal,
  signal,
} from '@angular/core';
import { MatTooltip } from '@angular/material/tooltip';
import {
  isActive,
  IsActiveMatchOptions,
  Router,
  RouterLink,
  RouterLinkActive,
} from '@angular/router';
import { TpTextButton } from '@team-platform/ui';
import { HasAccess } from '../../../core/directives/has-access';
import { RoleName } from '../../../core/enums/uam';
import { AuthService } from '../../../core/services/auth-service';

interface SideBarNavigationItem {
  label: string;
  hasAccess: boolean;
  icon: string;
  route: string | null;
  type: 'navigate' | 'dropdown';
  children: SideBarNavigationItem[];
}

@Component({
  selector: 'app-side-bar',
  imports: [RouterLink, RouterLinkActive, TpTextButton, HasAccess, MatTooltip],
  host: {
    '[class.tt-side-bar-host--collapsed]': 'collapsed()',
  },
  templateUrl: './side-bar.html',
  styleUrl: './side-bar.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SideBar {
  private readonly authService = inject(AuthService);
  private readonly router = inject(Router);
  readonly currentUser = computed(() => this.authService.currentUser());

  protected readonly expandedDropdownLabels = signal<ReadonlySet<string>>(new Set());
  private readonly childRouteMatchOptions: IsActiveMatchOptions = {
    paths: 'exact',
    queryParams: 'ignored',
    fragment: 'ignored',
    matrixParams: 'ignored',
  };
  private readonly childRouteActiveSignals = new Map<string, Signal<boolean>>();

  readonly collapsed = input(false);
  readonly expandRequested = output<void>();

  protected readonly navigationItems = computed<SideBarNavigationItem[]>(() => [
    {
      label: 'Home',
      hasAccess: true,
      icon: 'home',
      route: '/home',
      type: 'navigate',
      children: [],
    },
    {
      label: 'Admin Settings',
      icon: 'admin_panel_settings',
      hasAccess: this.hasAdminAccess(),
      route: null,
      type: 'dropdown',
      children: [
        {
          label: 'Users Management',
          icon: 'manage_accounts',
          hasAccess: this.hasAdminAccess(),
          route: '/admin-settings/users-management',
          type: 'navigate',
          children: [],
        },
      ],
    },
    {
      label: 'Settings',
      hasAccess: true,
      icon: 'settings',
      route: '/settings',
      type: 'navigate',
      children: [],
    },
  ]);

  private hasAdminAccess(): boolean {
    const roleName = this.currentUser()?.roleName;
    return roleName === RoleName.SuperAdmin || roleName === RoleName.Admin;
  }

  protected isDropdownExpanded(item: SideBarNavigationItem): boolean {
    return this.expandedDropdownLabels().has(item.label);
  }

  protected isDropdownActive(item: SideBarNavigationItem): boolean {
    return item.children.some((child) => child.route !== null && this.isRouteActive(child.route));
  }

  private isRouteActive(route: string): boolean {
    let routeActive = this.childRouteActiveSignals.get(route);

    if (!routeActive) {
      routeActive = isActive(route, this.router, this.childRouteMatchOptions);
      this.childRouteActiveSignals.set(route, routeActive);
    }

    return routeActive();
  }

  protected handleDropdownClick(item: SideBarNavigationItem): void {
    if (this.collapsed()) {
      this.expandedDropdownLabels.update((expandedLabels) =>
        new Set(expandedLabels).add(item.label),
      );
      this.expandRequested.emit();
      return;
    }

    this.toggleDropdown(item);
  }

  protected toggleDropdown(item: SideBarNavigationItem): void {
    this.expandedDropdownLabels.update((expandedLabels) => {
      const nextExpandedLabels = new Set(expandedLabels);

      if (nextExpandedLabels.has(item.label)) {
        nextExpandedLabels.delete(item.label);
      } else {
        nextExpandedLabels.add(item.label);
      }

      return nextExpandedLabels;
    });
  }
}
