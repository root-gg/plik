// Paste service
angular.module('paste', []).factory('$paste', function () {
    var module = {
        callback: null,
    };

    // Register a callback to execute when content is being pasted
    module.register = function (callback) {
        module.callback = callback
    };

    // Unregister the callback
    module.unregister = function () {
        module.callback = null;
    }

    // Paste event listener
    var pasteEventListener = function (event) {
        // Dismiss paste event if no callback is registered
        if (!module.callback) return;

        // Get the paste event clipboard data
        var clipboard = (event.clipboardData || window.clipboardData);
        module.callback(clipboard);
    };

    // Register paste event listener
    // This happens only once when the module is loaded
    window.addEventListener('paste', pasteEventListener);

    return module;
});