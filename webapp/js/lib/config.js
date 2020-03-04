// Config Service
angular.module('config', ['api']).factory('$config', function ($rootScope, $api) {
    var module = {
        config: $api.getConfig(),
        user: $api.getUser()
    };

    // Return config promise
    module.getConfig = function () {
        if (module.config) {
            return module.config;
        }
        return module.refreshConfig();
    };

    // Refresh config promise and notify listeners (top menu)
    module.refreshConfig = function () {
        module.config = $api.getConfig();
        $rootScope.$broadcast('config_refreshed', module.config);
        return module.config;
    };

    // Return user promise
    module.getUser = function () {
        if (module.user) {
            return module.user;
        }
        return module.refreshUser();
    };

    // Return original user promise
    module.getOriginalUser = function () {
        if (module.original_user) {
            return module.original_user;
        }
        module.refreshUser();
        return module.original_user;
    };

    // Refresh user promise and notify listeners (top menu)
    module.refreshUser = function () {
        module.user = $api.getUser();
        if (!module.original_user) {
            module.original_user = module.user;
        }
        $rootScope.$broadcast('user_refreshed', module.user);
        return module.user;
    };

    // Return server version
    module.getVersion = function () {
        if (module.version) {
            return module.version;
        }
        return module.refreshVersion()
    };

    // Refresh server version promise and notify listeners (top menu)
    module.refreshVersion = function () {
        module.version = $api.getVersion();
        $rootScope.$broadcast('version_refreshed', module.version);
        return module.version;
    };

    // Return server serverStats
    module.getServerStats = function () {
        if (module.serverStats) {
            return module.serverStats;
        }
        return module.refreshServerStats()
    };

    // Refresh server serverStats promise and notify listeners (top menu)
    module.refreshServerStats = function () {
        module.serverStats = $api.getServerStats();
        $rootScope.$broadcast('serverStats_refreshed', module.serverStats);
        return module.serverStats;
    };

    return module;
});