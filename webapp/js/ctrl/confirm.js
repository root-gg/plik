// Confirm controller
plik.controller('ConfirmCtrl', ['$scope', '$api', '$config', '$location', '$dialog',
    function ($scope, $api, $config, $location, $dialog) {

        // Get server config
        $config.getConfig()
            .then(function (config) {
                $scope.config = config;
                // Check if token authentication is enabled server side
                if (!config.authentication || config.registration === 'closed') {
                    $location.path('/');
                }
            })
            .then(null, function (error) {
                $dialog.alert(error);
            });

        // Get user from session
        $config.getUser()
            .then(function (user) {
                $scope.user = user;
            }, function (error) {
                if (error.status === 401 || error.status === 403) {
                    $location.path('/register');
                } else {
                    $dialog.alert(error);
                }
            });

        // Resend confirmation email
        $scope.resend = function () {
            $api.resend()
                .then(function (url) {
                    console.log(url);
                    $dialog.alert(url);
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };
    }]);