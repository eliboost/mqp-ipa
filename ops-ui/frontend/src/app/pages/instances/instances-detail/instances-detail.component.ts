import {Component, OnDestroy, OnInit, inject} from '@angular/core';
import {ShowQueuesComponent} from './show-queues/show-queues.component';
import {SharedModule} from '../../../shared/shared.module';
import {AMQInstance} from '../../../models/AMQInstance';
import {Subscription} from 'rxjs';
import {ActivatedRoute} from '@angular/router';
import {InstanceService} from '../../../services/instance.service';

@Component({
  selector: 'app-instances-detail',
  standalone: true,
  imports: [ShowQueuesComponent, SharedModule],
  templateUrl: './instances-detail.component.html',
  styleUrl: './instances-detail.component.scss',
})
export class InstancesDetailComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  instanceListService = inject(InstanceService);

  private paramSubscription: Subscription = new Subscription();
  private instanceSubscription: Subscription = new Subscription();

  public instance: AMQInstance = new AMQInstance();

  ngOnInit() {
    this.paramSubscription = this.route.paramMap.subscribe(queryParamMap => {
      const instanceNameFromQueryParameter = queryParamMap.get('name');
      const context = queryParamMap.get('context') || '';

      if (instanceNameFromQueryParameter) {
        this.instanceSubscription = this.instanceListService
          .getInstance(instanceNameFromQueryParameter, context)
          .subscribe({
            next: instance => {
              if (instance != null && instance.queues != null) {
                this.instance = instance;
              }
            },
          });
      }
    });
  }

  ngOnDestroy() {
    this.paramSubscription.unsubscribe();
    this.instanceSubscription.unsubscribe();
  }
}
