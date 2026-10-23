import {Injectable, inject} from '@angular/core';
import {lastValueFrom} from 'rxjs';
import {HttpClient} from '@angular/common/http';
import {EnvConfig} from '../../models/env-config';

@Injectable({
  providedIn: 'root',
})
export class EnvironmentLoaderService {
  private readonly http = inject(HttpClient);

  private envConfig!: EnvConfig;

  async loadEnvConfig(configPath: string): Promise<void> {
    this.envConfig = await lastValueFrom(this.http.get<EnvConfig>(configPath));
  }

  getEnvConfig(): EnvConfig {
    return this.envConfig;
  }
}
