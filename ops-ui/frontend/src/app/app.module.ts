import {HttpClient, HttpClientModule} from '@angular/common/http';
import {APP_INITIALIZER, NgModule} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {BrowserModule} from '@angular/platform-browser';
import {TranslateLoader, TranslateModule, TranslateService} from '@ngx-translate/core';
import {TranslateHttpLoader} from '@ngx-translate/http-loader';
import {PostFinanceComponentLibraryModule} from '@postfinance/component-library-angular';

import {AppRoutingModule} from './app-routing.module';
import {AppComponent} from './app.component';
import {NotFoundComponent} from './shared/not-found/not-found.component';
import {SharedModule} from './shared/shared.module';
import {getPfLocaleFrom} from '@postfinance/util-library';
import {map} from 'rxjs';
import {EnvironmentLoaderService} from './services/util/environment-loader.service';

// AoT requires an exported function for factories
export function HttpLoaderFactory(http: HttpClient) {
  return new TranslateHttpLoader(http, './assets/i18n/');
}

const initAppFn = (envService: EnvironmentLoaderService) => {
  return () => envService.loadEnvConfig('assets/config/app-config.json');
};

@NgModule({
  declarations: [AppComponent, NotFoundComponent],
  imports: [
    AppRoutingModule,
    BrowserModule,
    HttpClientModule,
    PostFinanceComponentLibraryModule.forRoot({
      i18n: {
        deps: [TranslateService],
        useFactory: (ngxTranslate: TranslateService) => ({
          translationHandler: (key, interpolateParams) => ngxTranslate.instant(key, interpolateParams),
          locale$: ngxTranslate.onLangChange.pipe(map(ev => getPfLocaleFrom(ev.lang))),
        }),
      },
    }),
    ReactiveFormsModule,
    SharedModule,
    TranslateModule.forRoot({
      defaultLanguage: 'de',
      loader: {
        provide: TranslateLoader,
        useFactory: HttpLoaderFactory,
        deps: [HttpClient],
      },
    }),
  ],
  providers: [
    EnvironmentLoaderService,
    {
      provide: APP_INITIALIZER,
      useFactory: initAppFn,
      multi: true,
      deps: [EnvironmentLoaderService],
    },
  ],
  bootstrap: [AppComponent],
})
export class AppModule {}
