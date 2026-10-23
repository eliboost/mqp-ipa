import {Component, enableProdMode, inject, OnDestroy, OnInit} from '@angular/core';
import {Router} from '@angular/router';
import {LangChangeEvent, TranslateService} from '@ngx-translate/core';
import {Subscription} from 'rxjs';
import {getPfLocaleFrom, PfEnvironment, pfLocalStorageLanguageKey} from '@postfinance/util-library';
import {
  PortalHeaderAppInfos,
  PortalHeaderMenuConfiguration,
  TitleAreaConfiguration,
  TitleAreaMode,
} from '@postfinance/component-library';
import {PfEnvironmentService} from '@postfinance/component-library-angular';
import {initAppInfos} from './app.infos';
import {EnvConfig} from './models/env-config';
import {EnvironmentLoaderService} from './services/util/environment-loader.service';
import {InstanceService} from './services/instance.service';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
  // eslint-disable-next-line @angular-eslint/prefer-standalone
  standalone: false,
})
export class AppComponent implements OnInit, OnDestroy {
  private router = inject(Router);
  private translate = inject(TranslateService);
  private pfEnvironmentService = inject(PfEnvironmentService);
  private readonly envService = inject(EnvironmentLoaderService);
  private instanceListService = inject(InstanceService);

  public envConfig?: EnvConfig;
  protected readonly PfEnvironment = PfEnvironment;
  protected readonly contexts$ = this.instanceListService.contexts$;

  // Meta-informations
  public environment?: PfEnvironment;
  public currentLocale: string;
  public appInfos: PortalHeaderAppInfos = initAppInfos();
  public version?: string;
  public displayEnv?: string;

  // Modify portal header
  public portalHeaderTitleAreaConfiguration: TitleAreaConfiguration = {
    mode: TitleAreaMode.NONE,
  };

  // Modify header menu
  public portalHeaderMenuConfiguration: Partial<PortalHeaderMenuConfiguration> = {
    hideLogout: false,
  };

  private subscription: Subscription = new Subscription();

  constructor() {
    const translate = this.translate;

    const defaultLanguage = this.getDefaultLanguage();
    this.setLanguage(localStorage.getItem(pfLocalStorageLanguageKey) || defaultLanguage);

    this.currentLocale = translate.currentLang.toUpperCase();

    this.envConfig = this.envService.getEnvConfig();
  }

  /**
   * Retrieves default language which is set in browser preferences.
   * Defaults to English when preference is not included in allowed languages.
   */
  public getDefaultLanguage = () => {
    const allowedLanguages = ['de', 'en'];
    const browserLanguage = navigator.language?.substring(0, 2);
    return allowedLanguages.find(lang => lang == browserLanguage) || 'de';
  };

  ngOnInit(): void {
    if (this.envConfig?.currentEnv === 'prod') {
      enableProdMode();
    }

    // eslint-disable-next-line @typescript-eslint/no-require-imports
    this.version = require('../../package.json').version;
    this.environment = this.envConfig?.currentEnv;
    this.displayEnv = this.envConfig?.displayEnv;

    this.subscription.add(
      this.translate.onLangChange.subscribe((event: LangChangeEvent) => {
        localStorage.setItem(pfLocalStorageLanguageKey, event.lang);
        const currentLocale = event.lang.toUpperCase();

        // parse the locale to OUI format:
        const pfLocale = getPfLocaleFrom(currentLocale);

        // notify OUI components of new locale:
        this.pfEnvironmentService.updateLanguage(pfLocale);
      }),
    );
  }

  public isActiveRoute(url: string, startsWith = false) {
    return startsWith ? this.router.url.startsWith(url) : this.router.url === url;
  }

  public setLanguage = (language: string) => {
    this.translate.use(language);
  };

  public setLanguageDe = () => this.setLanguage('de');

  public setLanguageEn = () => this.setLanguage('en');
  public isActiveLanguage = (language: string) => this.translate.currentLang === language;

  public goToInstance(context?: string) {
    this.router.navigate([context, 'instances']);
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
  }
}
