import { NgTemplateOutlet } from '@angular/common';
import type { TemplateRef } from '@angular/core';
import { ChangeDetectionStrategy, Component, computed, input, model } from '@angular/core';
import { TpDialog, TpDivider, TpTextButton } from '@team-platform/ui';

@Component({
  selector: 'app-title-dialog',
  imports: [NgTemplateOutlet, TpDialog, TpDivider, TpTextButton],
  templateUrl: './title-dialog.html',
  styleUrl: './title-dialog.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TitleDialog {
  open = model(false);
  ariaLabel = input('Dialog');
  title = input<string | TemplateRef<unknown>>('');
  bodyTemplate = input<TemplateRef<unknown> | null>(null);
  footerTemplate = input<TemplateRef<unknown> | null>(null);

  width = input('800px');
  maxWidth = input('1000px');
  height = input('auto');
  maxHeight = input('1000px');

  protected readonly titleTemplate = computed(() => {
    const title = this.title();
    return typeof title === 'string' ? null : title;
  });

  protected readonly titleText = computed(() => {
    const title = this.title();
    return typeof title === 'string' ? title : '';
  });

  protected close(): void {
    this.open.set(false);
  }
}
