import { Directive, TemplateRef, ViewContainerRef, effect, inject, input } from '@angular/core';

@Directive({
  selector: '[appHasAccess]',
})
export class HasAccess {
  readonly appHasAccess = input.required<boolean>();

  private readonly templateRef = inject(TemplateRef<unknown>);

  private readonly viewContainerRef = inject(ViewContainerRef);

  private hasView = false;

  constructor() {
    effect(() => {
      const hasAccess = this.appHasAccess();

      if (hasAccess && !this.hasView) {
        this.viewContainerRef.createEmbeddedView(this.templateRef);

        this.hasView = true;
      } else if (!hasAccess && this.hasView) {
        this.viewContainerRef.clear();
        this.hasView = false;
      }
    });
  }
}
