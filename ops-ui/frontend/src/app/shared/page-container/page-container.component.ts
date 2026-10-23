import {Component, Input} from '@angular/core';

@Component({
  selector: 'app-page-container',
  templateUrl: './page-container.component.html',
  styleUrls: ['./page-container.component.scss'],
  // eslint-disable-next-line @angular-eslint/prefer-standalone
  standalone: false,
})
export class PageContainerComponent {
  @Input()
  public pageTitle: string | undefined;
}
