import {Injectable, inject} from '@angular/core';
import {HttpClient} from '@angular/common/http';
import {EnvConfig} from '../models/env-config';
import {EnvironmentLoaderService} from './util/environment-loader.service';
import {Observable, shareReplay} from 'rxjs';
import {AMQInstance} from '../models/AMQInstance';
import {AMQContext} from '../models/AMQContext';

@Injectable({
  providedIn: 'root',
})
export class InstanceService {
  private http = inject(HttpClient);
  private readonly envService = inject(EnvironmentLoaderService);
  readonly contexts$ = this.http
    .get<AMQContext[]>(this.envService.getEnvConfig().apiUrl + '/v1/contexts')
    .pipe(shareReplay({bufferSize: 1, refCount: true}));
  private envConfig: EnvConfig;

  constructor() {
    this.envConfig = this.envService.getEnvConfig();
  }

  public getInstances(context: string): Observable<AMQInstance[]> {
    return this.http.get<AMQInstance[]>(this.envConfig.apiUrl + '/v1/instances?context=' + context);
  }

  public getInstance(instanceName: string, context: string): Observable<AMQInstance> {
    return this.http.get<AMQInstance>(
      this.envConfig.apiUrl + `/v1/instance?instanceToFind=${instanceName}&context=${context}`,
    );
  }
}
