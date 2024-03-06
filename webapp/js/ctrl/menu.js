plik.controller('MenuCtrl', ['$rootScope', '$scope', '$config',
    function ($rootScope, $scope, $config) {

        // Static config
        $rootScope.title = CONFIG.TITLE || "Plik";
        $scope.auth_button = !CONFIG.DISABLE_AUTH_BUTTON;

        // Get server config
        $config.getConfig()
            .then(function (config) {
                $scope.config = config;
            }, function () {
                // Avoid "Possibly unhandled rejection"
            });

        // Refresh config
        $rootScope.$on("config_refreshed", function (event, config) {
            config
                .then(function (c) {
                    $scope.config = c;
                })
                .then(null, function () {
                    $scope.config = null;
                });
        });

        $scope.isFeatureEnabled = function(feature_name) {
            if (!$scope.config) return false;
            var value = $scope.config["feature_" + feature_name]
            return value && value !== "disabled"
        }

        // Get user from session
        $config.getUser()
            .then(function (user) {
                $scope.user = user;
            }, function () {
                // Avoid "Possibly unhandled rejection"
            });

        // Refresh user
        $rootScope.$on("user_refreshed", function (event, user) {
            user
                .then(function (u) {
                    $scope.user = u;
                })
                .then(null, function () {
                    $scope.user = null;
                });
        });
    }]);