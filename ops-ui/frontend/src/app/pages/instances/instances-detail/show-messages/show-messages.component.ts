import {ChangeDetectorRef, Component, inject, OnDestroy, OnInit, Renderer2} from '@angular/core';
import {SharedModule} from '../../../../shared/shared.module';
import {PfDialogService, PostFinanceComponentLibraryModule} from '@postfinance/component-library-angular';
import {formatDate, NgIf} from '@angular/common';
import {ActivatedRoute} from '@angular/router';
import {
  ActionDialogTemplate,
  ButtonActionType,
  ButtonColor,
  ComboboxItemModel,
  DialogDisplayStateChangeEvent,
  HighlightJsLanguage,
  PfStatusSeverity,
  StatusBoxSize,
  TableCellContentData,
  TableFooterConfig,
  TableFooterMode,
  TableHeightMode,
  TableMenuConfiguration,
  TableMode,
  TableRowData,
  TableSelectedItemsChangedEvent,
} from '@postfinance/component-library';
import {forkJoin, Observable, Subscription} from 'rxjs';
import {AMQMessage, AMQMessageProperty} from '../../../../models/AMQMessage';
import {ExportEvent, QueueService, StreamingMessageEvent} from '../../../../services/queue.service';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {AMQInstance} from '../../../../models/AMQInstance';
import {AMQQueue} from '../../../../models/AMQQueue';
import {TranslateModule, TranslateService} from '@ngx-translate/core';
import {EnvConfig} from '../../../../models/env-config';
import {EnvironmentLoaderService} from '../../../../services/util/environment-loader.service';
import {InstanceService} from '../../../../services/instance.service';
import {AddMessageComponent} from '../add-message/add-message.component';
import {html_beautify} from 'js-beautify';

@Component({
  selector: 'app-show-messages',
  standalone: true,
  imports: [
    SharedModule,
    PostFinanceComponentLibraryModule,
    ReactiveFormsModule,
    TranslateModule,
    AddMessageComponent,
    NgIf,
  ],
  templateUrl: './show-messages.component.html',
  styleUrl: './show-messages.component.scss',
})
export class ShowMessagesComponent implements OnInit, OnDestroy {
  protected readonly TableMode = TableMode;
  protected readonly TableHeightMode = TableHeightMode;
  protected readonly PfStatusSeverity = PfStatusSeverity;
  protected readonly StatusBoxSize = StatusBoxSize;
  protected readonly ButtonColor = ButtonColor;
  protected tableMenuConfigMessagesTable: TableMenuConfiguration = {
    showActionsTitle: true,
  };
  protected dataForMessagesTableLoading = false;
  protected footerConfigMessagesTable: TableFooterConfig;
  protected selectedQueue: AMQQueue = new AMQQueue();
  protected messagesInSelectedQueue: AMQMessage[] = [];
  protected currentPageWhileFetchingMessages = 1;
  protected pageSizeWhileFetchingMessages = 200;
  protected selectedMessages: AMQMessage[] = [];
  protected selectedMessage: AMQMessage = new AMQMessage();
  protected showSelectAllMessage = false;
  protected allMessagesSelected = false;
  protected messageDetailsDialogOpen = false;
  protected messageBodyDialogOpen = false;
  protected possibleQueuesToMoveMessagesTo: ComboboxItemModel<string>[] = [];
  protected moveMessagesFormGroup = new FormGroup({
    newQueueNames: new FormControl<string>(''),
    blockSize: new FormControl<number>(1000, [Validators.required, Validators.min(1)]),
  });
  protected deleteBlockSizeControl = new FormControl<number>(1000, [Validators.required, Validators.min(1)]);
  protected moveMessagesDialogOpen = false;
  protected deleteMessagesDialogOpen = false;
  protected deleteAllMessagesDialogOpen = false;

  protected showErrorStatus = false;
  protected errorStatusText = '';
  protected showWarningStatus = false;
  protected warningStatusText = '';
  protected showSuccessStatus = false;
  protected successStatusText = '';

  protected bulkImportDialogOpen = false;

  protected actionInProgress = false;
  protected actionProgressProcessed = 0;
  protected actionProgressTotal = 0;

  protected instance: AMQInstance = new AMQInstance();
  private translate = inject(TranslateService);
  private queueService = inject(QueueService);
  private instanceService = inject(InstanceService);
  private dialogService = inject(PfDialogService);
  private readonly envService = inject(EnvironmentLoaderService);
  private readonly route = inject(ActivatedRoute);
  private readonly renderer = inject(Renderer2);
  private queueDetailSubscription: Subscription = new Subscription();
  private envConfig: EnvConfig;
  private cdr: ChangeDetectorRef = inject(ChangeDetectorRef);

  constructor() {
    this.footerConfigMessagesTable = this.refreshMessagesTableFooter();
    this.envConfig = this.envService.getEnvConfig();
    this.pageSizeWhileFetchingMessages = this.envConfig.pageSizeWhileFetchingMessages;
  }

  ngOnInit() {
    this.dataForMessagesTableLoading = true;

    const params = this.route.snapshot.params;
    this.instance.name = params['name'];
    this.instance.context = params['context'];
    const queueName: string = params['queue'];
    const replica = params['replica'] || 0;

    // Fetch the queue attributes so selectedQueue is fully populated
    this.fetchQueue(queueName, replica);

    // Fetch the full instance to populate the move-to combobox
    this.instanceService.getInstance(this.instance.name!, this.instance.context!).subscribe(instance => {
      this.instance = instance;
    });

    this.buildContextMenuEntriesMessagesTable();
  }

  public fetchQueue(queueName: string | undefined, replica: number | undefined) {
    if (!queueName) return;
    if (!replica) return;

    this.queueService
      .getQueueAttributes(this.instance.name, queueName, replica, this.instance.context)
      .subscribe(queue => {
        this.selectedQueue = queue;
        this.generateListOfMessagesInQueue();
      });
  }

  public generateListOfMessagesInQueue() {
    this.dataForMessagesTableLoading = true;
    this.possibleQueuesToMoveMessagesTo = [];
    this.messagesInSelectedQueue = [];
    this.currentPageWhileFetchingMessages = 1;

    this.filterPossibleQueues();

    this.queueDetailSubscription = this.queueService
      .getMessagesInQueue(
        this.instance.name,
        this.selectedQueue.name,
        this.selectedQueue.replica,
        String(this.currentPageWhileFetchingMessages),
        String(this.pageSizeWhileFetchingMessages),
        this.instance.context,
      )
      .subscribe({
        next: messages => {
          this.messagesInSelectedQueue = messages;
          this.dataForMessagesTableLoading = false;
        },
      });
  }

  public generateMoreMessagesIntoTheTable(page: number, pageSize: number) {
    this.dataForMessagesTableLoading = true;
    this.queueDetailSubscription = this.queueService
      .getMessagesInQueue(
        this.instance.name,
        this.selectedQueue.name,
        this.selectedQueue.replica,
        String(page),
        String(pageSize),
        this.instance.context,
      )
      .subscribe({
        next: messages => {
          this.messagesInSelectedQueue = [...this.messagesInSelectedQueue, ...messages];
          this.dataForMessagesTableLoading = false;
        },
      });

    this.cdr.markForCheck();
  }

  public formatTimestampToDateProvider = (
    _root: HTMLElement,
    _column: HTMLPfTableColumnElement,
    _rowData: TableRowData,
    cellValueProvider: () => string,
  ): TableCellContentData => {
    return {content: this.getFormattedDate(cellValueProvider()), isHtml: false};
  };

  public formatMessageTimestamp(timestamp: string | undefined) {
    return this.getFormattedDate(timestamp);
  }

  public getSelectedMessageProperties(): [string, AMQMessageProperty][] {
    return Object.entries(this.selectedMessage.properties ?? {}).sort((one, two) => (one > two ? -1 : 1));
  }

  public getSelectedMessageHeaders(): [string, string][] {
    return Object.entries(this.selectedMessage.headers ?? {}).sort((one, two) => (one > two ? 1 : -1));
  }

  public getBodyPreviewAsString(bodyPreview: number[] | undefined): string {
    if (!bodyPreview || bodyPreview.length === 0) {
      return '';
    }
    try {
      // Convert byte array to UTF-8 string
      return new TextDecoder().decode(new Uint8Array(bodyPreview));
    } catch {
      return '(Binary content - cannot display as text)';
    }
  }

  public getSelectedMessagesAsFormattedString() {
    return this.selectedMessages.map(m => m.messageID).join(', ');
  }

  public handleSelectedMessagesChange = (event: CustomEvent<TableSelectedItemsChangedEvent>) => {
    this.selectedMessages = [...event.detail.selectedItems];
    if (this.selectedMessages.length > 150) {
      this.showSelectAllMessage = true;
    }

    if (this.selectedMessages.length == 0) {
      this.showSelectAllMessage = false;
      this.allMessagesSelected = false;
    }
  };

  public handleMessageDetailsDialogStateChange = (event: CustomEvent<DialogDisplayStateChangeEvent>) => {
    this.messageDetailsDialogOpen = event.detail?.opened;
  };

  public handleMessageBodyDialogStateChange = (event: CustomEvent<DialogDisplayStateChangeEvent>) => {
    this.messageBodyDialogOpen = event.detail?.opened;
  };

  public handleScrolledToMessageTableBottomEvent = () => {
    this.generateMoreMessagesIntoTheTable(++this.currentPageWhileFetchingMessages, this.pageSizeWhileFetchingMessages);
  };

  ngOnDestroy() {
    this.queueDetailSubscription.unsubscribe();
  }

  public handleMessageAdded() {
    this.bulkImportDialogOpen = false;
    this.generateListOfMessagesInQueue();
  }

  public formatXml(input: string): string {
    if (!input) return '';

    try {
      return html_beautify(input, {indent_size: 2});
    } catch {
      return '';
    }
  }

  public formatContent(input: string): string {
    if (!input) return '';

    if (input.trim().startsWith('{')) {
      return JSON.stringify(JSON.parse(input), null, 2);
    } else if (input.trim().startsWith('<')) {
      return this.formatXml(input);
    }

    return '';
  }

  public determineCodeBlockLanguage(input: string): HighlightJsLanguage {
    if (!input) return 'json';

    if (input.trim().startsWith('{')) {
      return 'json';
    } else if (input.trim().startsWith('<')) {
      return 'xml';
    }

    return 'json';
  }

  protected exportSelectedMessages() {
    if (this.selectedMessages.length > 0) {
      const dialogTemplate: ActionDialogTemplate<ButtonActionType.CONFIRM | ButtonActionType.CANCEL> = {
        buttons: [ButtonActionType.CONFIRM, ButtonActionType.CANCEL],
        titleTextKey: this.translate.instant('INSTANCES.DIALOGS.EXPORT_MESSAGE_CONFIRM.TITLE'),
        messageKey: this.translate.instant('INSTANCES.DIALOGS.EXPORT_MESSAGE_CONFIRM.MESSAGE', {
          messages: this.allMessagesSelected
            ? this.translate.instant('INSTANCES.SHOW_MESSAGES.ALL_MESSAGES')
            : this.selectedMessages.map(m => m.messageID).join(', '),
        }),
      };

      const selectedQueueName = this.selectedQueue.name;
      const selectedQueueReplica = this.selectedQueue.replica;
      if (selectedQueueName === null || selectedQueueName === undefined) return;
      if (selectedQueueReplica === null || selectedQueueReplica === undefined) return;

      this.dialogService.showActionDialogCancellableObservable(dialogTemplate).subscribe(resp => {
        if (resp === ButtonActionType.CONFIRM) {
          this.dataForMessagesTableLoading = true;
          this.hideAllTextBoxes();
          this.actionInProgress = true;
          this.actionProgressProcessed = 0;
          this.actionProgressTotal = this.allMessagesSelected
            ? (this.selectedQueue.messageCount ?? 0)
            : this.selectedMessages.length;

          const stream$ = this.allMessagesSelected
            ? this.queueService.exportAllMessages(
                this.instance.name ?? '',
                this.selectedQueue.name ?? '',
                this.selectedQueue.replica ?? 0,
                this.instance.context ?? '',
                this.selectedQueue.messageCount ?? 0,
              )
            : this.queueService.exportMessages(
                this.instance.name ?? '',
                this.selectedQueue.name ?? '',
                this.selectedQueue.replica ?? 0,
                this.instance.context ?? '',
                this.selectedMessages.map(m => m.messageID ?? ''),
              );

          stream$.subscribe({
            next: (event: ExportEvent) => {
              if (event.type === 'progress') {
                this.actionProgressProcessed = event.processed ?? 0;
                this.actionProgressTotal = event.total ?? this.actionProgressTotal;
              } else if (event.type === 'complete') {
                this.triggerDownload(event.content!, event.filename!);
              } else if (event.type === 'error') {
                this.showErrorTextBox('INSTANCES.STATUS_BOXES.EXPORT_MESSAGES.ERROR');
              }
            },
            error: () => {
              this.showErrorTextBox('INSTANCES.STATUS_BOXES.EXPORT_MESSAGES.ERROR');
              this.resetActionTextBox();
              this.generateListOfMessagesInQueue();
              this.dataForMessagesTableLoading = false;
            },
            complete: () => {
              this.resetActionTextBox();
              if (!this.showWarningStatus && !this.showErrorStatus) {
                this.showSuccessTextBox('INSTANCES.STATUS_BOXES.EXPORT_MESSAGES.SUCCESS');
              }
              this.generateListOfMessagesInQueue();
              this.dataForMessagesTableLoading = false;
              this.fetchQueue(selectedQueueName, selectedQueueReplica);
            },
          });
        }
      });
    }
  }

  protected moveSelectedMessages() {
    if (this.selectedMessages.length > 0 && this.moveMessagesFormGroup.valid) {
      const dialogTemplate: ActionDialogTemplate<ButtonActionType.CONFIRM | ButtonActionType.CANCEL> = {
        buttons: [ButtonActionType.CONFIRM, ButtonActionType.CANCEL],
        titleTextKey: this.translate.instant('INSTANCES.DIALOGS.MOVE_MESSAGES_CONFIRM.TITLE'),
        messageKey: this.translate.instant('INSTANCES.DIALOGS.MOVE_MESSAGES_CONFIRM.MESSAGE', {
          messages: this.allMessagesSelected
            ? this.translate.instant('INSTANCES.SHOW_MESSAGES.ALL_MESSAGES')
            : this.selectedMessages.map(m => m.messageID).join(', '),
          targetQueue: this.moveMessagesFormGroup.get('newQueueNames')?.value,
        }),
      };

      const selectedQueueName = this.selectedQueue.name;
      const selectedQueueReplica = this.selectedQueue.replica;
      if (selectedQueueName === null || selectedQueueName === undefined) return;
      if (selectedQueueReplica === null || selectedQueueReplica === undefined) return;

      this.dialogService.showActionDialogCancellableObservable(dialogTemplate).subscribe(resp => {
        if (resp === ButtonActionType.CONFIRM) {
          this.dataForMessagesTableLoading = true;

          this.hideAllTextBoxes();
          this.actionInProgress = true;
          this.closeMoveMessagesDialog();

          const stream$ = this.allMessagesSelected
            ? this.queueService.moveAllMessages(
                this.instance.name ?? '',
                this.selectedQueue.name ?? '',
                this.moveMessagesFormGroup.get('newQueueNames')?.value ?? '',
                this.selectedQueue.replica ?? 0,
                this.instance.context ?? '',
                this.moveMessagesFormGroup.controls.blockSize.value ?? 0,
              )
            : this.queueService.moveMessages(
                this.instance.name ?? '',
                this.selectedQueue.name ?? '',
                this.moveMessagesFormGroup.get('newQueueNames')?.value ?? '',
                this.selectedQueue.replica ?? 0,
                this.instance.context ?? '',
                this.selectedMessages.map(m => m.messageID ?? ''),
              );

          stream$.subscribe({
            next: (event: StreamingMessageEvent) => {
              if (event.type === 'progress') {
                this.actionProgressProcessed = event.processed ?? 0;
                if (event.total == -1) {
                  this.actionProgressTotal = this.selectedQueue.messageCount ?? 0;
                } else {
                  this.actionProgressTotal = this.selectedQueue.messageCount ?? 0;
                }
              } else if (event.type === 'complete') {
                if ((event.failed ?? 0) > 0) {
                  this.showWarningTextBox('INSTANCES.STATUS_BOXES.MOVE_MESSAGES.WARNING');
                } else {
                  this.showSuccessTextBox('INSTANCES.STATUS_BOXES.MOVE_MESSAGES.SUCCESS');
                }
              } else if (event.type === 'error') {
                this.showErrorTextBox('INSTANCES.STATUS_BOXES.MOVE_MESSAGES.ERROR');
              }
            },
            error: () => {
              this.showErrorTextBox('INSTANCES.STATUS_BOXES.MOVE_MESSAGES.ERROR');
              this.generateListOfMessagesInQueue();
              this.dataForMessagesTableLoading = false;
              this.closeMoveMessagesDialog();
            },
            complete: () => {
              if (!this.showWarningStatus && !this.showErrorStatus) {
                this.showSuccessTextBox('INSTANCES.STATUS_BOXES.MOVE_MESSAGES.SUCCESS');
              }
              this.generateListOfMessagesInQueue();
              this.dataForMessagesTableLoading = false;
              this.closeMoveMessagesDialog();
              this.resetActionTextBox();
            },
          });
        }
      });
    }
  }

  protected deleteSelectedMessages() {
    if (this.selectedMessages.length === 0) return;

    const selectedQueueName = this.selectedQueue.name;
    const selectedQueueReplica = this.selectedQueue.replica;
    if (selectedQueueName === null || selectedQueueName === undefined) return;
    if (selectedQueueReplica === null || selectedQueueReplica === undefined) return;

    const dialogTemplate: ActionDialogTemplate<ButtonActionType.DELETE | ButtonActionType.CANCEL> = {
      buttons: [ButtonActionType.DELETE, ButtonActionType.CANCEL],
      titleTextKey: this.translate.instant('INSTANCES.DIALOGS.DELETE_MESSAGES_CONFIRM.TITLE'),
      messageKey: this.translate.instant('INSTANCES.DIALOGS.DELETE_MESSAGES_CONFIRM.MESSAGE', {
        messages: this.allMessagesSelected
          ? this.translate.instant('INSTANCES.SHOW_MESSAGES.ALL_MESSAGES')
          : this.selectedMessages.map(m => m.messageID).join(', '),
      }),
    };

    this.dialogService.showActionDialogCancellableObservable(dialogTemplate).subscribe(resp => {
      if (resp === ButtonActionType.DELETE) {
        this.dataForMessagesTableLoading = true;
        this.deleteMessagesDialogOpen = false;
        this.hideAllTextBoxes();
        this.actionInProgress = true;

        const stream$ = this.allMessagesSelected
          ? this.queueService.deleteAllMessages(
              this.instance.name || '',
              selectedQueueName,
              selectedQueueReplica,
              this.instance.context || '',
              this.deleteBlockSizeControl.value || 0,
            )
          : this.queueService.deleteMessages(
              this.instance.name || '',
              selectedQueueName,
              selectedQueueReplica,
              this.instance.context || '',
              this.selectedMessages.map(m => m.messageID ?? ''),
            );

        stream$.subscribe({
          next: (event: StreamingMessageEvent) => {
            if (event.type === 'progress') {
              this.actionProgressProcessed = event.processed ?? 0;
              if (event.total == -1) {
                this.actionProgressTotal = this.selectedQueue.messageCount ?? 0;
              } else {
                this.actionProgressTotal = this.selectedQueue.messageCount ?? 0;
              }
            } else if (event.type === 'complete') {
              if ((event.failed ?? 0) > 0) {
                this.showWarningTextBox('INSTANCES.STATUS_BOXES.DELETE_MESSAGES.WARNING');
              } else {
                this.showSuccessTextBox('INSTANCES.STATUS_BOXES.DELETE_MESSAGES.SUCCESS');
              }
            } else if (event.type === 'error') {
              this.showErrorTextBox('INSTANCES.STATUS_BOXES.DELETE_MESSAGES.ERROR');
            }
          },
          error: () => {
            this.showErrorTextBox('INSTANCES.STATUS_BOXES.DELETE_MESSAGES.ERROR');
            this.generateListOfMessagesInQueue();
            this.dataForMessagesTableLoading = false;
          },
          complete: () => {
            if (!this.showWarningStatus && !this.showErrorStatus) {
              this.showSuccessTextBox('INSTANCES.STATUS_BOXES.DELETE_MESSAGES.SUCCESS');
            }
            this.generateListOfMessagesInQueue();
            this.resetActionTextBox();
            this.dataForMessagesTableLoading = false;
          },
        });
      }
    });
  }

  protected showErrorTextBox(translationKey: string) {
    this.showErrorStatus = true;
    this.errorStatusText = this.translate.instant(translationKey);
  }

  protected selectAllMessages = () => (this.allMessagesSelected = true);

  protected undoSelectAllMessages = () => (this.allMessagesSelected = false);

  protected openMessageDetailsDialog = () => (this.messageDetailsDialogOpen = true);

  protected openMessageBodyDialog = () => (this.messageBodyDialogOpen = true);

  protected openMoveMessagesDialog = () => (this.moveMessagesDialogOpen = true);

  protected openDeleteMessageDialog = () => (this.deleteMessagesDialogOpen = true);

  protected closeMoveMessagesDialog = () => (this.moveMessagesDialogOpen = false);

  protected recommendedBlockSize(messageCount: number): number {
    return Math.min(Math.max(Math.round(messageCount / 10), 2), 5000);
  }

  protected showSuccessTextBox(translationKey: string) {
    this.showSuccessStatus = true;
    this.successStatusText = this.translate.instant(translationKey);
  }

  protected hideAllTextBoxes() {
    this.showSuccessStatus = false;
    this.showWarningStatus = false;
    this.showErrorStatus = false;
  }

  protected showWarningTextBox(translationKey: string) {
    this.showWarningStatus = true;
    this.warningStatusText = this.translate.instant(translationKey);
  }

  private triggerDownload(base64Content: string, filename: string) {
    const binary = atob(base64Content);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    const blob = new Blob([bytes], {type: 'application/xml'});
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  private filterPossibleQueues() {
    if (!this.selectedQueue.name) return;

    const parentQueue: string = this.selectedQueue.name.replace('_EXPIRY', '').replace('_DLQ', '');

    this.instance.queues
      ?.filter(s => s !== this.selectedQueue.name && s.startsWith(parentQueue))
      .forEach(q => {
        const menuItem: ComboboxItemModel<string> = {label: q, value: q};
        this.possibleQueuesToMoveMessagesTo.push(menuItem);
      });
  }

  private buildContextMenuEntriesMessagesTable = () => {
    this.tableMenuConfigMessagesTable.headerMenuItems = [
      {
        identifier: 'move',
        icon: 'sl:direction-button-arrows',
        label: this.translate.instant('INSTANCES.MESSAGES_TABLE.BUTTON_MOVE'),
        action: () => {
          if (this.selectedMessages.length > 0) {
            this.openMoveMessagesDialog();
          }
        },
        disabled: () => this.selectedMessages.length == 0,
      },
      {
        identifier: 'delete',
        icon: 'sl:bin-1',
        label: this.translate.instant('INSTANCES.MESSAGES_TABLE.BUTTON_DELETE'),
        action: () => {
          if (this.selectedMessages.length > 0) {
            this.openDeleteMessageDialog();
          }
        },
        disabled: () => this.selectedMessages.length == 0,
      },
      {
        identifier: 'export',
        icon: 'sl:common-file-download',
        label: 'Export',
        action: () => this.exportSelectedMessages(),
        disabled: () => this.selectedMessages.length == 0,
      },
    ];

    this.tableMenuConfigMessagesTable.headerToolbarItems = [
      {
        identifier: 'import',
        icon: 'sl:upload-bottom',
        label: 'Import messages',
        action: () => (this.bulkImportDialogOpen = !this.bulkImportDialogOpen),
      },
      {
        identifier: 'recover',
        icon: 'sl:keyboard-arrow-return',
        label: this.translate.instant('INSTANCES.MESSAGES_TABLE.BUTTON_RECOVER'),
        action: () => this.recoverSelectedMessages(),
        hidden: () => !/(_DLQ|_EXPIRY)$/.test(this.selectedQueue?.name ?? ''),
      },
    ];

    this.tableMenuConfigMessagesTable.rowToolbarItems = [
      {
        identifier: 'detail',
        icon: 'view-1',
        label: this.translate.instant('INSTANCES.MESSAGES_TABLE.BUTTON_DETAILS'),
        action: (rowData: TableRowData<AMQMessage>) => {
          this.onShowMessageDetailsClick(rowData.item);
        },
        showLabel: true,
        tooltipText: () => {
          return this.translate.instant('INSTANCES.MESSAGES_TABLE.BUTTON_DETAILS_TOOLTIP');
        },
      },
      {
        identifier: 'message-body',
        icon: 'view-1',
        label: this.translate.instant('INSTANCES.MESSAGES_TABLE.BUTTON_MESSAGE_BODY'),
        action: (rowData: TableRowData<AMQMessage>) => {
          this.onShowMessageBodyClick(rowData.item);
        },
        showLabel: true,
        tooltipText: () => {
          return this.translate.instant('INSTANCES.MESSAGES_TABLE.BUTTON_MESSAGE_BODY_TOOLTIP');
        },
      },
    ];
  };

  private getFormattedDate(timestamp: string | undefined) {
    let result: string;
    if (timestamp === undefined || timestamp === null) {
      result = '';
    } else {
      result = formatDate(timestamp, 'dd.MM.yyyy, HH:mm:ss.SSS', 'en-US');
    }
    return result;
  }

  private onShowMessageDetailsClick = (amqMessage: AMQMessage) => {
    this.selectedMessage = amqMessage;
    this.openMessageDetailsDialog();
  };

  private onShowMessageBodyClick = (amqMessage: AMQMessage) => {
    this.selectedMessage = amqMessage;
    this.openMessageBodyDialog();
  };

  private refreshMessagesTableFooter = (): TableFooterConfig => {
    return {
      mode: TableFooterMode.STATUS,
      showRowCount: true,
    };
  };

  private recoverSelectedMessages() {
    if (this.selectedMessages.length > 0) {
      const dialogTemplate: ActionDialogTemplate<ButtonActionType.CONFIRM | ButtonActionType.CANCEL> = {
        buttons: [ButtonActionType.CONFIRM, ButtonActionType.CANCEL],
        titleTextKey: this.translate.instant('INSTANCES.DIALOGS.RECOVER_MESSAGES_CONFIRM.TITLE'),
        messageKey: this.translate.instant('INSTANCES.DIALOGS.RECOVER_MESSAGES_CONFIRM.MESSAGE', {
          messages: this.selectedMessages.map(m => m.messageID).join(', '),
        }),
      };

      this.dialogService.showActionDialogCancellableObservable(dialogTemplate).subscribe(resp => {
        if (resp === ButtonActionType.CONFIRM) {
          this.dataForMessagesTableLoading = true;

          this.hideAllTextBoxes();

          const messageObservables: Observable<boolean>[] = [];

          this.selectedMessages.forEach(message => {
            if (
              message.properties?.['_AMQ_ORIG_QUEUE'].value !== undefined ||
              message.properties?.['_AMQ_ORIG_QUEUE'].value !== ''
            ) {
              const recoverSingleMessageMethodCall = this.queueService.recoverSingleMessage(
                this.instance.name,
                this.selectedQueue.name,
                message.messageID,
                this.selectedQueue.replica,
                this.instance.context,
              );

              messageObservables.push(recoverSingleMessageMethodCall);
            } else {
              this.showWarningTextBox('INSTANCES.STATUS_BOXES.RECOVER_MESSAGES.WARNING');
            }
          });

          forkJoin(messageObservables).subscribe({
            next: value => {
              if (value.includes(false) && value.includes(true)) {
                this.showWarningTextBox('INSTANCES.STATUS_BOXES.RECOVER_MESSAGES.WARNING');
              } else if (value.includes(false) && !value.includes(true)) {
                this.showErrorTextBox('INSTANCES.STATUS_BOXES.RECOVER_MESSAGES.ERROR');
              }
            },
            complete: () => {
              if (!this.showWarningStatus || !this.showErrorStatus) {
                this.showSuccessTextBox('INSTANCES.STATUS_BOXES.RECOVER_MESSAGES.SUCCESS');
              }
              this.generateListOfMessagesInQueue();
              this.dataForMessagesTableLoading = false;
            },
          });

          if (messageObservables.length == 0) {
            this.showErrorTextBox('INSTANCES.STATUS_BOXES.RECOVER_MESSAGES.ERROR');
          }
        }
      });
    }
  }

  private resetActionTextBox() {
    this.actionInProgress = false;
    this.actionProgressTotal = 0;
    this.actionProgressProcessed = 0;
  }
}
