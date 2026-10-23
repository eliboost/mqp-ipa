import {NgModule} from '@angular/core';
import {CommonModule} from '@angular/common';

import {InstancesRoutingModule} from './instances-routing.module';
import {PostFinanceComponentLibraryModule} from '@postfinance/component-library-angular';
import {SharedModule} from '../../shared/shared.module';
import {ReactiveFormsModule} from '@angular/forms';
import {InstancesComponent} from './instances-list/instances.component';
import {ShowQueuesComponent} from './instances-detail/show-queues/show-queues.component';
import {ShowMessagesComponent} from './instances-detail/show-messages/show-messages.component';

@NgModule({
  declarations: [],
  imports: [
    CommonModule,
    InstancesRoutingModule,
    PostFinanceComponentLibraryModule,
    SharedModule,
    ReactiveFormsModule,
    InstancesComponent,
    ShowQueuesComponent,
    ShowMessagesComponent,
  ],
})
export class InstancesModule {}
