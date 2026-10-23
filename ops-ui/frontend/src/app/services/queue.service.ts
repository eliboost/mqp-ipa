import {inject, Injectable, NgZone} from '@angular/core';
import {HttpClient, HttpHeaders} from '@angular/common/http';
import {EnvConfig} from '../models/env-config';
import {EnvironmentLoaderService} from './util/environment-loader.service';
import {Observable} from 'rxjs';
import {AMQMessage} from '../models/AMQMessage';
import {AMQQueue} from '../models/AMQQueue';

export interface ExportEvent {
  type: 'progress' | 'complete' | 'error';
  processed?: number;
  total?: number;
  filename?: string;
  content?: string;
  message?: string;
}

export interface ImportEvent {
  type: 'progress' | 'complete' | 'error';
  done?: number;
  total?: number;
  failed?: number;
  failedFilename?: string;
  failedContent?: string;
  message?: string;
}

export interface StreamingMessageEvent {
  type: 'progress' | 'complete' | 'error';
  processed?: number;
  total?: number;
  failed?: number;
  message?: string;
}

export type QueueStateOperation = 'pause' | 'resume' | 'enable' | 'disable';

@Injectable({
  providedIn: 'root',
})
export class QueueService {
  private http = inject(HttpClient);
  private readonly envService = inject(EnvironmentLoaderService);
  private readonly ngZone = inject(NgZone);

  private envConfig: EnvConfig;

  constructor() {
    this.envConfig = this.envService.getEnvConfig();
  }

  public getQueueAttributes(
    instanceName?: string,
    queueName?: string,
    replica?: number,
    context?: string,
  ): Observable<AMQQueue> {
    return this.http.get<AMQQueue>(
      this.envConfig.apiUrl +
        '/v1/queue/attributes?instanceName=' +
        instanceName +
        '&queueName=' +
        queueName +
        '&replica=' +
        replica +
        '&context=' +
        context,
    );
  }

  public getMessagesInQueue(
    instanceName?: string,
    queueName?: string,
    replica?: number,
    page?: string,
    pageSize?: string,
    context?: string,
  ): Observable<AMQMessage[]> {
    return this.http.get<AMQMessage[]>(
      this.envConfig.apiUrl +
        '/v1/queue/messagesInQueue?instanceName=' +
        instanceName +
        '&queueName=' +
        queueName +
        '&replica=' +
        replica +
        '&page=' +
        page +
        '&pageSize=' +
        pageSize +
        '&context=' +
        context,
    );
  }

  public recoverSingleMessage(
    instanceName?: string,
    queueName?: string,
    messageId?: string,
    replica?: number,
    context?: string,
  ): Observable<boolean> {
    const headers = new HttpHeaders().set('Content-Type', 'application/json');
    return this.http.post<boolean>(
      this.envConfig.apiUrl +
        '/v1/queue/recoverMessage?instanceName=' +
        instanceName +
        '&queueName=' +
        queueName +
        '&replica=' +
        replica +
        '&context=' +
        context,
      messageId,
      {headers},
    );
  }

  public deleteMessages(
    instanceName: string,
    queueName: string,
    replica: number,
    context: string,
    messageIds: string[],
  ): Observable<StreamingMessageEvent> {
    return this.streamRequest('/v1/queue/deleteMessages?instanceName=' + instanceName + '&context=' + context, {
      instanceName,
      queueName,
      replica,
      context,
      messageIds,
    });
  }

  public deleteAllMessages(
    instanceName: string,
    queueName: string,
    replica: number,
    context: string,
    blockSize: number,
  ): Observable<StreamingMessageEvent> {
    return this.streamRequest('/v1/queue/deleteAllMessages?instanceName=' + instanceName + '&context=' + context, {
      instanceName,
      queueName,
      replica,
      context,
      blockSize,
    });
  }

  public exportMessages(
    instanceName: string,
    queueName: string,
    replica: number,
    context: string,
    messageIds: string[],
  ): Observable<ExportEvent> {
    return this.streamExport('/v1/queue/exportMessages?instanceName=' + instanceName + '&context=' + context, {
      queueName,
      replica,
      messageIds,
    });
  }

  public exportAllMessages(
    instanceName: string,
    queueName: string,
    replica: number,
    context: string,
    totalCount: number,
  ): Observable<ExportEvent> {
    return this.streamExport('/v1/queue/exportAllMessages?instanceName=' + instanceName + '&context=' + context, {
      queueName,
      replica,
      totalCount,
    });
  }

  public importMessages(
    instanceName: string,
    queueName: string,
    replica: number,
    context: string,
    journalFile: File | File[],
  ): Observable<ImportEvent> {
    const file = Array.isArray(journalFile) ? journalFile[0] : journalFile;
    const formData = new FormData();
    formData.append('file', file, file.name);

    const path = `/v1/queue/importMessages?instanceName=${instanceName}&queueName=${queueName}&replica=${replica}&context=${context}`;
    return this.streamMultipartNdjson(path, formData);
  }

  public importWeblogicMessages(
    instanceName: string,
    queueName: string,
    replica: number,
    context: string,
    journalFile: File | File[],
  ): Observable<ImportEvent> {
    const file = Array.isArray(journalFile) ? journalFile[0] : journalFile;
    const formData = new FormData();
    formData.append('file', file, file.name);

    const path = `/v1/queue/importWeblogicMessages?instanceName=${instanceName}&queueName=${queueName}&replica=${replica}&context=${context}`;
    return this.streamMultipartNdjson(path, formData);
  }

  public changeQueueState(
    instanceName?: string,
    queueName?: string,
    operation?: QueueStateOperation,
    replica?: number,
    context?: string,
  ): Observable<boolean> {
    const headers = new HttpHeaders().set('Content-Type', 'application/json');
    return this.http.post<boolean>(
      this.envConfig.apiUrl +
        '/v1/queue/changeQueueState?instanceName=' +
        instanceName +
        '&queueName=' +
        queueName +
        '&replica=' +
        replica +
        '&context=' +
        context,
      {operation},
      {headers},
    );
  }

  public moveMessages(
    instanceName: string,
    currentQueueName: string,
    newQueueName: string,
    replica: number,
    context: string,
    messageIds: string[],
  ): Observable<StreamingMessageEvent> {
    return this.streamRequest('/v1/queue/moveMessages?instanceName=' + instanceName + '&context=' + context, {
      currentQueueName,
      newQueueName,
      replica,
      messageIds,
    });
  }

  public moveAllMessages(
    instanceName: string,
    currentQueueName: string,
    newQueueName: string,
    replica: number,
    context: string,
    blockSize: number,
  ): Observable<StreamingMessageEvent> {
    return this.streamRequest('/v1/queue/moveAllMessages?instanceName=' + instanceName + '&context=' + context, {
      currentQueueName,
      newQueueName,
      replica,
      blockSize,
    });
  }

  private streamRequest(path: string, body: object): Observable<StreamingMessageEvent> {
    return new Observable<StreamingMessageEvent>(observer => {
      const controller = new AbortController();

      fetch(this.envConfig.apiUrl + path, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(body),
        credentials: 'include',
        signal: controller.signal,
      })
        .then(async response => {
          if (!response.ok || !response.body) {
            this.ngZone.run(() => observer.error(new Error('Delete request failed')));
            return;
          }

          const reader = response.body.getReader();
          const decoder = new TextDecoder();
          let buffer = '';

          while (true) {
            const {done, value} = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, {stream: true});

            const lines = buffer.split('\n');
            buffer = lines.pop() ?? '';

            for (const line of lines) {
              const trimmed = line.trim();
              if (!trimmed) continue;
              try {
                const event = JSON.parse(trimmed) as StreamingMessageEvent;
                this.ngZone.run(() => observer.next(event));
              } catch {
                // ignore malformed lines
              }
            }
          }
          this.ngZone.run(() => observer.complete());
        })
        .catch(err => {
          this.ngZone.run(() => {
            if (err.name !== 'AbortError') observer.error(err);
            else observer.complete();
          });
        });

      return () => controller.abort();
    });
  }

  private streamExport(path: string, body: object): Observable<ExportEvent> {
    return new Observable<ExportEvent>(observer => {
      const controller = new AbortController();

      fetch(this.envConfig.apiUrl + path, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(body),
        credentials: 'include',
        signal: controller.signal,
      })
        .then(async response => {
          if (!response.ok || !response.body) {
            this.ngZone.run(() => observer.error(new Error('Export request failed')));
            return;
          }

          const reader = response.body.getReader();
          const decoder = new TextDecoder();
          let buffer = '';

          while (true) {
            const {done, value} = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, {stream: true});

            const lines = buffer.split('\n');
            buffer = lines.pop() ?? '';

            for (const line of lines) {
              const trimmed = line.trim();
              if (!trimmed) continue;
              try {
                const event = JSON.parse(trimmed) as ExportEvent;
                this.ngZone.run(() => observer.next(event));
              } catch {
                // ignore malformed lines
              }
            }
          }
          this.ngZone.run(() => observer.complete());
        })
        .catch(err => {
          this.ngZone.run(() => {
            if (err.name !== 'AbortError') observer.error(err);
            else observer.complete();
          });
        });

      return () => controller.abort();
    });
  }

  private streamMultipartNdjson(path: string, body: FormData): Observable<ImportEvent> {
    return new Observable<ImportEvent>(observer => {
      const controller = new AbortController();

      fetch(this.envConfig.apiUrl + path, {
        method: 'POST',
        body,
        credentials: 'include',
        signal: controller.signal,
      })
        .then(async response => {
          if (!response.ok || !response.body) {
            this.ngZone.run(() => observer.error(new Error('Import request failed')));
            return;
          }

          const reader = response.body.getReader();
          const decoder = new TextDecoder();
          let buffer = '';

          while (true) {
            const {done, value} = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, {stream: true});

            const lines = buffer.split('\n');
            buffer = lines.pop() ?? '';

            for (const line of lines) {
              const trimmed = line.trim();
              if (!trimmed) continue;
              try {
                const event = JSON.parse(trimmed) as ImportEvent;
                this.ngZone.run(() => observer.next(event));
              } catch {
                // ignore malformed lines
              }
            }
          }
          this.ngZone.run(() => observer.complete());
        })
        .catch(err => {
          this.ngZone.run(() => {
            if (err.name !== 'AbortError') observer.error(err);
            else observer.complete();
          });
        });

      return () => controller.abort();
    });
  }
}
