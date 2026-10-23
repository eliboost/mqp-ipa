import {Component, inject, OnInit} from '@angular/core';
import {Router} from '@angular/router';
import {InstanceService} from '../services/instance.service';
import {firstValueFrom} from 'rxjs';
import {EnvironmentLoaderService} from '../services/util/environment-loader.service';
import {EnvConfig} from '../models/env-config';

@Component({
  selector: 'app-dynamic-redirect',
  template: '',
  standalone: true,
})
export class DynamicRedirectComponent implements OnInit {
  private router = inject(Router);
  private instanceListService = inject(InstanceService);
  private readonly envService = inject(EnvironmentLoaderService);
  public envConfig?: EnvConfig;

  constructor() {
    this.envConfig = this.envService.getEnvConfig();
  }

  async ngOnInit(): Promise<void> {
    const contexts = await firstValueFrom(this.instanceListService.contexts$);
    const first = contexts?.[0];
    if (first) {
      this.router.navigate(['/', first.name, 'instances']);
    }
  }
}
