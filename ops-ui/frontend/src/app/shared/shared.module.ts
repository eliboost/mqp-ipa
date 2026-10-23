import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {PostFinanceComponentLibraryModule} from '@postfinance/component-library-angular';
import {PageContainerComponent} from './page-container/page-container.component';

@NgModule({
  declarations: [PageContainerComponent],
  exports: [PageContainerComponent],
  imports: [CommonModule, PostFinanceComponentLibraryModule],
})
export class SharedModule {}
