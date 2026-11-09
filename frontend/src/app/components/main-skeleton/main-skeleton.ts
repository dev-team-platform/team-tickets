import { ChangeDetectionStrategy, Component } from '@angular/core';
import { TpSkeleton } from '@team-platform/ui';

@Component({
  selector: 'app-main-skeleton',
  imports: [TpSkeleton],
  templateUrl: './main-skeleton.html',
  styleUrl: './main-skeleton.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MainSkeleton {
  protected readonly featureActions = [0, 1, 2];
  protected readonly tableFilters = [0, 1, 2, 3];
  protected readonly tableRows = [
    ['78%', '54%', '68%', '42%'],
    ['62%', '84%', '48%', '72%'],
    ['88%', '46%', '74%', '58%'],
    ['52%', '66%', '88%', '44%'],
    ['72%', '58%', '52%', '82%'],
    ['46%', '76%', '62%', '68%'],
  ];
}
