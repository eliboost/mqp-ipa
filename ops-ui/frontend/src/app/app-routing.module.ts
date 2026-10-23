import {NgModule} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {NotFoundComponent} from './shared/not-found/not-found.component';
import {DynamicRedirectComponent} from './pages/dynamic-redirect';

const routes: Routes = [
  {path: '', component: DynamicRedirectComponent, pathMatch: 'full'},
  {path: 'instances', component: DynamicRedirectComponent, pathMatch: 'full'},
  {
    path: ':context',
    children: [
      {
        path: 'instances',
        loadChildren: () => import('./pages/instances/instances.module').then(m => m.InstancesModule),
      },
    ],
  },
  // {
  //   path: 'instances-detail',
  //   loadChildren: () =>
  //     import('./pages/instances/instances-detail/instances-detail.module').then(m => m.InstancesModule),
  // },
  {path: '**', component: NotFoundComponent},
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule],
})
export class AppRoutingModule {}
