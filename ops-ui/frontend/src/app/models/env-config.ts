import {PfEnvironment} from '@postfinance/util-library';

export interface EnvConfig {
  production: boolean;
  apiUrl: string;
  currentEnv: PfEnvironment;
  displayEnv: string;
  allowSendMessages: boolean;
  pageSizeWhileFetchingMessages: number;
}
