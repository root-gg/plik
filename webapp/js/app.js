// Plik app bootstrap and global configuration
var plik = angular.module('plik', ['ngRoute', 'api', 'config', 'dialog', 'paste', 'contentEditable', 'btford.markdown'])
    .config(function ($routeProvider) {
        $routeProvider
            .when('/', {controller: 'MainCtrl', templateUrl: 'partials/main.html', reloadOnSearch: false})
            .when('/clients', {controller: 'ClientListCtrl', templateUrl: 'partials/clients.html'})
            .when('/login', {controller: 'LoginCtrl', templateUrl: 'partials/login.html'})
            .when('/home', {controller: 'HomeCtrl', templateUrl: 'partials/home.html'})
            .when('/admin', {controller: 'AdminCtrl', templateUrl: 'partials/admin.html'})
            .otherwise({redirectTo: '/'});
    })
    .config(['$locationProvider', function ($locationProvider) {
        // see https://github.com/angular/angular.js/commit/aa077e81129c740041438688dff2e8d20c3d7b52
        // see https://webmasters.googleblog.com/2015/10/deprecating-our-ajax-crawling-scheme.html
        $locationProvider.hashPrefix("");
    }])
    .config(['$httpProvider', function ($httpProvider) {
        $httpProvider.defaults.headers.common['X-ClientApp'] = 'web_client';
        $httpProvider.defaults.xsrfCookieName = 'plik-xsrf';
        $httpProvider.defaults.xsrfHeaderName = 'X-XSRFToken';

        // Mangle "Connection failed" result for alert modal
        $httpProvider.interceptors.push(function ($q) {
            return {
                responseError: function (resp) {
                    if (resp.status <= 0) {
                        resp.data = {status: resp.status, message: "Connection failed"};
                    }
                    return $q.reject(resp);
                }
            };
        });
    }])
    .filter('collapseClass', function () {
        return function (opened) {
            if (opened) return "fa fa-caret-down";
            return "fa fa-caret-right";
        }
    });

new ClipboardJS('[data-clipboard]')
