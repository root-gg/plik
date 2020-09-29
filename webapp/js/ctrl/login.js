// Login controller
plik.controller('LoginCtrl', ['$scope', '$api', '$config', '$location', '$dialog',
    function ($scope, $api, $config, $location, $dialog) {

        // Ugly but it works
        setTimeout(function () {
            $("#login").focus();
        }, 100);

        // Get server config
        $config.getConfig()
            .then(function (config) {
                $scope.config = config;
                // Check if token authentication is enabled server side
                if (!config.authentication) {
                    $location.path('/');
                }
            })
            .then(null, function (error) {
                if (error.status !== 401 && error.status !== 403) {
                    $dialog.alert(error);
                }
            });

        // Get user from session
        $config.getUser()
            .then(function () {
                $location.path('/home');
            })
            .then(null, function (error) {
                if (error.status !== 401 && error.status !== 403) {
                    $dialog.alert(error);
                }
            });

        // Google authentication
        $scope.google = function () {
            $api.login("google")
                .then(function (url) {
                    // Redirect to Google user consent dialog
                    window.location.replace(url);
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // OVH authentication
        $scope.ovh = function () {
            $api.login("ovh")
                .then(function (url) {
                    // Redirect to OVH user consent dialog
                    window.location.replace(url);
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // Login with local user
        $scope.login = function () {
            $api.login("local", $scope.username, $scope.password)
                .then(function () {
                    $config.refreshUser();
                    $location.path('/home');
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };
    }]);