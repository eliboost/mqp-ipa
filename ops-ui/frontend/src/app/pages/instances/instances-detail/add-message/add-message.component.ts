import {
  ChangeDetectionStrategy,
  ChangeDetectorRef,
  Component,
  EventEmitter,
  inject,
  Input,
  Output,
} from '@angular/core';
import {PostFinanceComponentLibraryModule} from '@postfinance/component-library-angular';
import {ImportEvent, QueueService} from '../../../../services/queue.service';
import {AMQInstance} from '../../../../models/AMQInstance';
import {AMQQueue} from '../../../../models/AMQQueue';
import {DialogDisplayStateChangeEvent, FileUploadModes, FormLayoutOrientation} from '@postfinance/component-library';
import {FormControl, FormGroup, FormsModule, ReactiveFormsModule, Validators} from '@angular/forms';
import {TranslateModule} from '@ngx-translate/core';

@Component({
  selector: 'app-add-message',
  standalone: true,
  imports: [PostFinanceComponentLibraryModule, ReactiveFormsModule, TranslateModule, FormsModule],
  templateUrl: './add-message.component.html',
  styleUrl: './add-message.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AddMessageComponent {
  @Input() instance: AMQInstance = new AMQInstance();
  @Input() queue: AMQQueue = new AMQQueue();
  @Output() messageAdded = new EventEmitter<void>();
  @Output() errorMessage = new EventEmitter<string>();
  @Output() successMessage = new EventEmitter<string>();
  @Output() submitting = new EventEmitter<void>();
  @Output() importComplete = new EventEmitter<void>();
  public bulkImportMessagesForm = new FormGroup({
    journalFile: new FormControl<File | null>(null, control => Validators.required(control)),
  });
  public weblogicImportMessageForm = new FormGroup({
    file: new FormControl<File | null>(null, control => Validators.required(control)),
  });
  protected readonly FileUploadModes = FileUploadModes;
  protected readonly FormLayoutOrientation = FormLayoutOrientation;
  protected bulkImportDialogOpen = false;

  protected isSubmitting = false;
  protected failed = false;

  protected bulkImportDone = 0;
  protected bulkImportTotal = 0;
  protected bulkImportFailed = 0;
  protected showBulkImportProgress = false;

  private queueService = inject(QueueService);
  private cdr = inject(ChangeDetectorRef);

  @Input() set opened(value: boolean) {
    this.bulkImportDialogOpen = value;
  }

  @Input() set bulkImportOpen(value: boolean) {
    this.bulkImportDialogOpen = value;
  }

  public onSubmitBulkImport(importType: string) {
    this.bulkImportMessagesForm.markAllAsTouched();

    this.showBulkImportProgress = true;
    this.isSubmitting = true;
    this.submitting.emit();

    this.bulkImportDone = 0;
    this.bulkImportTotal = 0;
    this.bulkImportFailed = 0;
    this.cdr.markForCheck();

    const stream$ =
      importType == 'artemis'
        ? this.queueService.importMessages(
            this.instance.name || '',
            this.queue.name || '',
            this.queue.replica || 0,
            this.instance.context || '',
            this.bulkImportMessagesForm.controls.journalFile.value as File,
          )
        : this.queueService.importWeblogicMessages(
            this.instance.name || '',
            this.queue.name || '',
            this.queue.replica || 0,
            this.instance.context || '',
            this.weblogicImportMessageForm.controls.file.value as File,
          );

    stream$.subscribe({
      next: (event: ImportEvent) => {
        if (event.type === 'progress') {
          this.bulkImportDone = event.done ?? this.bulkImportDone;
          this.bulkImportTotal = event.total ?? this.bulkImportTotal;
          this.bulkImportFailed = event.failed ?? this.bulkImportFailed;
          this.cdr.markForCheck();
          return;
        }

        if (event.type === 'complete') {
          this.bulkImportDone = event.done ?? this.bulkImportDone;
          this.bulkImportTotal = event.total ?? this.bulkImportTotal;
          this.bulkImportFailed = event.failed ?? this.bulkImportFailed;
          this.cdr.markForCheck(); // TODO: a good reason why we should migrate to signals
          if (event.failedContent && event.failedFilename) {
            this.triggerDownload(event.failedContent, event.failedFilename, 'application/xml');
          }
        }

        if (event.type === 'error') {
          this.emitError();
          this.cdr.markForCheck();
        }
      },
      error: () => {
        this.isSubmitting = false;
        this.emitError();
        this.cdr.markForCheck();
      },
      complete: () => {
        this.isSubmitting = false;
        if (!this.failed) {
          this.successMessage.emit('INSTANCES.STATUS_BOXES.ADD_MESSAGE.BULK_IMPORT_SUCCESS');
          this.bulkImportMessagesForm.reset();
          this.messageAdded.emit();
        }
        this.cdr.markForCheck();
        this.importComplete.emit();
      },
    });
  }

  public isInvalid(control: FormControl): boolean {
    return control.invalid && control.touched;
  }

  public handleAddMessageDialogStateChange = (event: CustomEvent<DialogDisplayStateChangeEvent>) => {
    if (!event.detail?.opened) {
      this.closeDialog();
    }
  };

  protected closeDialog() {
    this.bulkImportDialogOpen = false;
  }

  private triggerDownload(base64Content: string, filename: string, mimeType: string) {
    const binary = atob(base64Content);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    const blob = new Blob([bytes], {type: mimeType});
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  private emitError() {
    this.errorMessage.emit('INSTANCES.STATUS_BOXES.ADD_MESSAGE.BULK_IMPORT_ERROR');
    this.failed = true;
  }
}
