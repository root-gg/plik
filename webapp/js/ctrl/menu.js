plik.controller('MenuCtrl', ['$rootScope', '$scope', '$config',
    function ($rootScope, $scope, $config) {

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