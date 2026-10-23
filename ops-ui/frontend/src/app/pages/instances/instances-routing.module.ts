import {NgModule} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {InstancesComponent} from './instances-list/instances.component';
import {InstancesDetailComponent} from './instances-detail/instances-detail.component';
import {ShowMessagesComponent} from './instances-detail/show-messages/show-messages.component';

const routes: Routes = [
  {path: '', component: InstancesComponent},
  {
    path: 'instance-detail',
    component: InstancesDetailComponent,
  },
  {
    path: 'messages',
    component: ShowMessagesComponent,
  },
  // {path: ':name', component: InstancesDetailComponent},
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule],
})
export class InstancesRoutingModule {}
