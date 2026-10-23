import {Component, inject, Input, OnDestroy, OnInit} from '@angular/core';
import {
  ActionDialogTemplate,
  ButtonActionType,
  MenuItemColor,
  PfAuxiliaryStatus,
  TableCellContentData,
  TableHeightMode,
  TableMenuConfiguration,
  TableMode,
  TableRowData,
  TableSelectedItemsChangedEvent,
  TableSize,
} from '@postfinance/component-library';
import {AMQQueue} from '../../../../models/AMQQueue';
import {QueueService} from '../../../../services/queue.service';
import {forkJoin, Observable, Subscription} from 'rxjs';
import {PfDialogService, PostFinanceComponentLibraryModule} from '@postfinance/component-library-angular';
import {AMQInstance} from '../../../../models/AMQInstance';
import {ReactiveFormsModule} from '@angular/forms';
import {SharedModule} from '../../../../shared/shared.module';
import {TranslateModule, TranslateService} from '@ngx-translate/core';
import {Router} from '@angular/router';
import {TableHeaderDropdownMenuItems} from '@postfinance/component-library/dist/types/components/data-display/table/table.interface';

@Component({
  selector: 'app-show-queues',
  standalone: true,
  imports: [PostFinanceComponentLibraryModule, SharedModule, ReactiveFormsModule, TranslateModule],
  templateUrl: './show-queues.component.html',
  styleUrl: './show-queues.component.scss',
})
export class ShowQueuesComponent implements OnInit, OnDestroy {
  protected readonly TableMode = TableMode;
  protected readonly TableHeightMode = TableHeightMode;
  protected tableMenuConfigQueuesTable: TableMenuConfiguration = {
    showActionsTitle: true,
  };
  protected queuesForThisInstance: AMQQueue[] = [];
  protected selectedQueues: AMQQueue[] = [];
  protected dataForQueuesTableLoading = false;
  protected readonly TableSize = TableSize;
  protected addMessageDialogOpen = false;

  private translate = inject(TranslateService);
  private queueService = inject(QueueService);
  private dialogService = inject(PfDialogService);
  private readonly router = inject(Router);
  private headerMenuItems: TableHeaderDropdownMenuItems = [
    {
      identifier: 'enable',
      icon: 'sl:launch-go-flag',
      label: this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_ENABLE'),
      action: () => this.enableQueue(),
      color: MenuItemColor.SECONDARY,
    },
    {
      identifier: 'disable',
      icon: 'sl:disable',
      label: this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_DISABLE'),
      action: () => this.disableQueue(),
      color: MenuItemColor.SECONDARY,
    },
    {
      identifier: 'pause',
      icon: 'sl:button-pause-1',
      label: this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_PAUSE'),
      action: () => this.pauseQueue(),
      color: MenuItemColor.SECONDARY,
    },
    {
      identifier: 'resume',
      icon: 'sl:button-play-1',
      label: this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_RESUME'),
      action: () => this.resumeQueue(),
      color: MenuItemColor.SECONDARY,
    },
  ];
  private queueDetailSubscription: Subscription = new Subscription();

  constructor() {
    this.buildContextMenuEntriesQueuesTable();
  }

  private _instance: AMQInstance = new AMQInstance();

  get instance(): AMQInstance {
    return this._instance;
  }

  @Input() set instance(value: AMQInstance) {
    this._instance = value;
    if (value?.queues != null) {
      this.generateDetailsForQueues();
    }
  }

  ngOnInit() {
    if (this.instance?.queues != null) {
      this.generateDetailsForQueues();
    } else {
      this.dataForQueuesTableLoading = false;
    }
  }

  public renderEnabledChip = (
    root: HTMLElement,
    column: HTMLPfTableColumnElement,
    rowData: TableRowData,
  ): TableCellContentData => {
    let content: string;
    if (rowData.item?.enabled) {
      content =
        '<pf-status-chip status="' +
        PfAuxiliaryStatus.GREEN +
        '">' +
        this.translate.instant('INSTANCES.QUEUES_TABLE.COLUMN_TITLE_ENABLED_YES') +
        '</pf-status-chip>';
    } else {
      content =
        '<pf-status-chip status="' +
        PfAuxiliaryStatus.RED +
        '">' +
        this.translate.instant('INSTANCES.QUEUES_TABLE.COLUMN_TITLE_ENABLED_NO') +
        '</pf-status-chip>';
    }
    return {content, isHtml: true};
  };

  public renderPausedChip = (
    root: HTMLElement,
    column: HTMLPfTableColumnElement,
    rowData: TableRowData,
  ): TableCellContentData => {
    let content: string;
    if (rowData.item?.paused) {
      content =
        '<pf-status-chip status="' +
        PfAuxiliaryStatus.RED +
        '">' +
        this.translate.instant('INSTANCES.QUEUES_TABLE.COLUMN_TITLE_PAUSED_YES') +
        '</pf-status-chip>';
    } else {
      content =
        '<pf-status-chip status="' +
        PfAuxiliaryStatus.GREEN +
        '">' +
        this.translate.instant('INSTANCES.QUEUES_TABLE.COLUMN_TITLE_PAUSED_NO') +
        '</pf-status-chip>';
    }
    return {content, isHtml: true};
  };

  public handleSelectedQueuesChange = (event: CustomEvent<TableSelectedItemsChangedEvent>) => {
    this.selectedQueues = [...event.detail.selectedItems];

    this.tableMenuConfigQueuesTable = {
      ...this.tableMenuConfigQueuesTable,
      headerMenuItems: this.selectedQueues.length > 0 ? this.headerMenuItems : [],
    };
  };

  ngOnDestroy() {
    this.queueDetailSubscription.unsubscribe();
    this.queuesForThisInstance = [];
  }

  public handleAddMessageDialogClosed() {
    this.addMessageDialogOpen = false;
  }

  public handleMessageAdded() {
    this.generateDetailsForQueues();
  }

  // Create list of all queues for this instance and the number of messages in them
  private generateDetailsForQueues() {
    this.dataForQueuesTableLoading = true;
    const queueObservables: Observable<AMQQueue>[] = [];

    if (this.instance != null && this.instance.queues != null) {
      this.instance.queues.forEach(queueName => {
        if (this.instance.replicas !== undefined) {
          for (let i = 0; i < this.instance.replicas; i++) {
            const queueObservable = this.queueService.getQueueAttributes(
              this.instance.name,
              queueName,
              i,
              this.instance.context,
            );
            queueObservables.push(queueObservable);
          }
        }
      });

      const tempAllQueuesForThisInstance: AMQQueue[] = [];

      forkJoin(queueObservables).subscribe({
        next: (queues: AMQQueue[]) => {
          queues.forEach(queue => {
            tempAllQueuesForThisInstance.push(queue);
          });
        },
        complete: () => {
          this.queuesForThisInstance = tempAllQueuesForThisInstance;
          this.dataForQueuesTableLoading = false;
        },
      });
    } else {
      this.dataForQueuesTableLoading = false;
    }
  }

  private buildContextMenuEntriesQueuesTable = () => {
    this.tableMenuConfigQueuesTable.rowToolbarItems = [
      {
        identifier: 'detail',
        icon: 'view-1',
        label: this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_SHOW_MESSAGES'),
        action: (rowData: TableRowData<AMQQueue>) => {
          this.router.navigate([
            this.instance.context,
            'instances',
            'messages',
            {
              name: this.instance.name,
              context: this.instance.context,
              queue: rowData.item.name,
              replica: rowData.item.replica,
            },
          ]);
        },
        color: MenuItemColor.SECONDARY,
        showLabel: true,
        tooltipText: () => {
          return this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_SHOW_MESSAGES_TOOLTIP');
        },
      },

      // Actions
      {
        identifier: 'enable',
        icon: 'sl:launch-go-flag',
        label: this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_ENABLE'),
        showLabel: true,
        action: (rowData: TableRowData<AMQQueue>) => {
          this.selectedQueues = [rowData.item];
          this.enableQueue();
        },
        color: MenuItemColor.SECONDARY,
        hidden: (rowData: TableRowData<AMQQueue>) => rowData.item.enabled!,
      },
      {
        identifier: 'disable',
        icon: 'sl:disable',
        label: this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_DISABLE'),
        showLabel: true,
        tooltipText: () => {
          return this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_SHOW_MESSAGES_TOOLTIP');
        },
        action: (rowData: TableRowData<AMQQueue>) => {
          this.selectedQueues = [rowData.item];
          this.disableQueue();
        },
        color: MenuItemColor.SECONDARY,
        hidden: (rowData: TableRowData<AMQQueue>) => !rowData.item.enabled!,
      },
      {
        identifier: 'pause',
        icon: 'sl:button-pause-1',
        label: this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_PAUSE'),
        showLabel: true,
        tooltipText: () => {
          return this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_SHOW_MESSAGES_TOOLTIP');
        },
        action: (rowData: TableRowData<AMQQueue>) => {
          this.selectedQueues = [rowData.item];
          this.pauseQueue();
        },
        color: MenuItemColor.SECONDARY,
        hidden: (rowData: TableRowData<AMQQueue>) => rowData.item.paused!,
      },
      {
        identifier: 'resume',
        icon: 'sl:button-play-1',
        label: this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_RESUME'),
        showLabel: true,
        tooltipText: () => {
          return this.translate.instant('INSTANCES.QUEUES_TABLE.BUTTON_SHOW_MESSAGES_TOOLTIP');
        },
        action: (rowData: TableRowData<AMQQueue>) => {
          this.selectedQueues = [rowData.item];
          this.resumeQueue();
        },
        color: MenuItemColor.SECONDARY,
        hidden: (rowData: TableRowData<AMQQueue>) => !rowData.item.paused!,
      },
    ];

    this.tableMenuConfigQueuesTable.headerMenuItems = [];
  };

  private enableQueue() {
    if (this.selectedQueues.length > 0) {
      const dialogTemplate: ActionDialogTemplate<ButtonActionType.CONFIRM | ButtonActionType.CANCEL> = {
        buttons: [ButtonActionType.CONFIRM, ButtonActionType.CANCEL],
        titleTextKey: this.translate.instant('INSTANCES.DIALOGS.ENABLE_QUEUE.TITLE'),
        messageKey: this.translate.instant('INSTANCES.DIALOGS.ENABLE_QUEUE.MESSAGE', {
          queues: this.selectedQueues.map(q => q.name).join(', '),
        }),
      };

      this.dialogService.showActionDialogCancellableObservable(dialogTemplate).subscribe(resp => {
        if (resp === ButtonActionType.CONFIRM) {
          this.dataForQueuesTableLoading = true;

          const enableQueueObservables: Observable<boolean>[] = [];

          this.selectedQueues.forEach(queue => {
            const enableQueueMethodCall = this.queueService.changeQueueState(
              this.instance.name,
              queue.name,
              'enable',
              queue.replica,
              this.instance.context,
            );

            enableQueueObservables.push(enableQueueMethodCall);
          });

          forkJoin(enableQueueObservables).subscribe({
            complete: () => {
              this.generateDetailsForQueues();
            },
          });
        }
      });
    }
  }

  private disableQueue() {
    if (this.selectedQueues.length > 0) {
      const dialogTemplate: ActionDialogTemplate<ButtonActionType.CONFIRM | ButtonActionType.CANCEL> = {
        buttons: [ButtonActionType.CONFIRM, ButtonActionType.CANCEL],
        titleTextKey: this.translate.instant('INSTANCES.DIALOGS.DISABLE_QUEUE.TITLE'),
        messageKey: this.translate.instant('INSTANCES.DIALOGS.DISABLE_QUEUE.MESSAGE', {
          queues: this.selectedQueues.map(q => q.name).join(', '),
        }),
      };

      this.dialogService.showActionDialogCancellableObservable(dialogTemplate).subscribe(resp => {
        if (resp === ButtonActionType.CONFIRM) {
          this.dataForQueuesTableLoading = true;

          const enableQueueObservables: Observable<boolean>[] = [];

          this.selectedQueues.forEach(queue => {
            const enableQueueMethodCall = this.queueService.changeQueueState(
              this.instance.name,
              queue.name,
              'disable',
              queue.replica,
              this.instance.context,
            );

            enableQueueObservables.push(enableQueueMethodCall);
          });

          forkJoin(enableQueueObservables).subscribe({
            complete: () => {
              this.generateDetailsForQueues();
            },
          });
        }
      });
    }
  }

  private pauseQueue() {
    if (this.selectedQueues.length > 0) {
      const dialogTemplate: ActionDialogTemplate<ButtonActionType.CONFIRM | ButtonActionType.CANCEL> = {
        buttons: [ButtonActionType.CONFIRM, ButtonActionType.CANCEL],
        titleTextKey: this.translate.instant('INSTANCES.DIALOGS.PAUSE_QUEUE.TITLE'),
        messageKey: this.translate.instant('INSTANCES.DIALOGS.PAUSE_QUEUE.MESSAGE', {
          queues: this.selectedQueues.map(q => q.name).join(', '),
        }),
      };

      this.dialogService.showActionDialogCancellableObservable(dialogTemplate).subscribe(resp => {
        if (resp === ButtonActionType.CONFIRM) {
          this.dataForQueuesTableLoading = true;

          const pauseQueueObservables: Observable<boolean>[] = [];

          this.selectedQueues.forEach(queue => {
            const pauseQueueMethodCall = this.queueService.changeQueueState(
              this.instance.name,
              queue.name,
              'pause',
              queue.replica,
              this.instance.context,
            );

            pauseQueueObservables.push(pauseQueueMethodCall);
          });

          forkJoin(pauseQueueObservables).subscribe({
            complete: () => {
              this.generateDetailsForQueues();
            },
          });
        }
      });
    }
  }

  private resumeQueue() {
    if (this.selectedQueues.length > 0) {
      const dialogTemplate: ActionDialogTemplate<ButtonActionType.CONFIRM | ButtonActionType.CANCEL> = {
        buttons: [ButtonActionType.CONFIRM, ButtonActionType.CANCEL],
        titleTextKey: this.translate.instant('INSTANCES.DIALOGS.RESUME_QUEUE.TITLE'),
        messageKey: this.translate.instant('INSTANCES.DIALOGS.RESUME_QUEUE.MESSAGE', {
          queues: this.selectedQueues.map(q => q.name).join(', '),
        }),
      };

      this.dialogService.showActionDialogCancellableObservable(dialogTemplate).subscribe(resp => {
        if (resp === ButtonActionType.CONFIRM) {
          this.dataForQueuesTableLoading = true;

          const resumeQueueObservables: Observable<boolean>[] = [];

          this.selectedQueues.forEach(queue => {
            const resumeQueueMethodCall = this.queueService.changeQueueState(
              this.instance.name,
              queue.name,
              'resume',
              queue.replica,
              this.instance.context,
            );

            resumeQueueObservables.push(resumeQueueMethodCall);
          });

          forkJoin(resumeQueueObservables).subscribe({
            complete: () => {
              this.generateDetailsForQueues();
            },
          });
        }
      });
    }
  }
}
