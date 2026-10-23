import {Component, OnInit, inject} from '@angular/core';
import {InstanceService} from '../../../services/instance.service';
import {AMQInstance} from '../../../models/AMQInstance';
import {
  PfAuxiliaryStatus,
  PfStatusSeverity,
  StatusBoxSize,
  TableCellContentData,
  TableHeightMode,
  TableMode,
  TableRowData,
  TableSelectedRowNavigationEvent,
  TextColumnTextFlow,
} from '@postfinance/component-library';
import {ActivatedRoute, Router} from '@angular/router';
import {PostFinanceComponentLibraryModule} from '@postfinance/component-library-angular';
import {SharedModule} from '../../../shared/shared.module';
import {TranslateModule, TranslateService} from '@ngx-translate/core';
import {Subscription} from 'rxjs';

@Component({
  selector: 'app-instances',
  standalone: true,
  templateUrl: './instances.component.html',
  styleUrls: ['./instances.component.scss'],
  imports: [PostFinanceComponentLibraryModule, SharedModule, TranslateModule],
})
export class InstancesComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private translate = inject(TranslateService);
  private router = inject(Router);
  private instanceListService = inject(InstanceService);

  protected readonly TextColumnTextFlow = TextColumnTextFlow;
  protected readonly TableMode = TableMode;
  protected readonly TableHeightMode = TableHeightMode;

  protected readonly StatusBoxSize = StatusBoxSize;
  protected readonly PfStatusSeverity = PfStatusSeverity;

  protected dataForTableIsLoading = false;
  protected instances: AMQInstance[] = [];
  protected context: string | undefined;
  protected selectedInstance: AMQInstance = new AMQInstance();

  protected showInstanceOfflineErrorStatusBox = false;

  private paramSubscription: Subscription = new Subscription();

  ngOnInit(): void {
    this.dataForTableIsLoading = true;

    // Get the name of the current instance from the query parameter
    this.paramSubscription = this.route.paramMap.subscribe(queryParamMap => {
      const contextNameFromQueryParameter = queryParamMap.get('context');
      if (contextNameFromQueryParameter) {
        this.context = contextNameFromQueryParameter;
        // Get all instances from backend
        this.instanceListService.getInstances(contextNameFromQueryParameter).subscribe({
          next: instances => {
            this.instances = instances;
            this.dataForTableIsLoading = false;
          },
        });
      }
    });
  }

  // Returns online or offline chip in instances table
  public renderOnlineChip = (
    _root: HTMLElement,
    _column: HTMLPfTableColumnElement,
    _rowData: TableRowData,
  ): TableCellContentData => {
    let content: string;
    if (_rowData.item?.online) {
      content =
        '<pf-status-chip status="' +
        PfAuxiliaryStatus.GREEN +
        '">' +
        this.translate.instant('INSTANCES.INSTANCES_TABLE.STATE_CHIP_ONLINE') +
        '</pf-status-chip>';
    } else {
      content =
        '<pf-status-chip status="' +
        PfAuxiliaryStatus.RED +
        '">' +
        this.translate.instant('INSTANCES.INSTANCES_TABLE.STATE_CHIP_OFFLINE') +
        '</pf-status-chip>';
    }
    return {content, isHtml: true};
  };

  public instanceRowSelectedEvent(event: CustomEvent<TableSelectedRowNavigationEvent>) {
    this.selectedInstance = event.detail.item;

    // Navigate to details of an instance if it is online
    if (this.selectedInstance.online) {
      //this.router.navigate(['instances/instance-detail', {name: this.selectedInstance.name}]);
      this.router.navigate([
        this.context,
        'instances',
        'instance-detail',
        {name: this.selectedInstance.name, context: this.context},
      ]);
    } else {
      this.showInstanceOfflineErrorStatusBox = true;
    }
  }
}
