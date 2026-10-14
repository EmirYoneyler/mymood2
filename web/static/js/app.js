// mymood — small progressive enhancements. Every page works without this
// file; it only adds live feedback. Listeners are delegated on document so
// they keep working across htmx-boosted page swaps.
(function () {
    'use strict';

    // Same stops as scoreGradientStops in internal/handlers/calendar.go.
    var STOPS = [
        [1.0, [255, 59, 48]],
        [3.25, [255, 140, 26]],
        [5.5, [255, 214, 10]],
        [6.6, [163, 230, 53]],
        [7.75, [34, 197, 94]],
        [10.0, [6, 214, 199]]
    ];

    function scoreColor(score) {
        if (score <= STOPS[0][0]) return rgb(STOPS[0][1]);
        for (var i = 1; i < STOPS.length; i++) {
            if (score <= STOPS[i][0]) {
                var a = STOPS[i - 1], b = STOPS[i];
                var t = (score - a[0]) / (b[0] - a[0]);
                return rgb([0, 1, 2].map(function (k) {
                    return Math.round(a[1][k] + (b[1][k] - a[1][k]) * t);
                }));
            }
        }
        return rgb(STOPS[STOPS.length - 1][1]);
    }

    function rgb(c) {
        return 'rgb(' + c[0] + ',' + c[1] + ',' + c[2] + ')';
    }

    function bucket(score) {
        if (score >= 9) return 'excellent';
        if (score >= 7) return 'great';
        if (score >= 5) return 'good';
        if (score >= 3) return 'low';
        return 'poor';
    }

    // --- Score picker (slider + number stay in sync, tag shows the colour) ---

    function syncPicker(root, raw, source) {
        var range = root.querySelector('[data-score-range]');
        var input = root.querySelector('[data-score-input]');
        var tag = root.querySelector('[data-score-tag]');
        var value = parseFloat(String(raw).replace(',', '.'));
        if (isNaN(value)) return;
        value = Math.min(10, Math.max(1, Math.round(value * 10) / 10));

        if (source !== 'range') range.value = value;
        if (source !== 'input') input.value = value.toFixed(1);
        fitWidth(input);
        if (tag) {
            tag.style.backgroundColor = scoreColor(value);
            tag.textContent = tag.getAttribute('data-label-' + bucket(value)) || tag.textContent;
        }
    }

    // Shrink the big number input to its text so "/10" sits right next to it.
    function fitWidth(input) {
        var probe = document.createElement('span');
        probe.className = input.className + ' picker__measure';
        probe.textContent = input.value || '0.0';
        input.parentNode.appendChild(probe);
        input.style.width = Math.ceil(probe.getBoundingClientRect().width) + 8 + 'px';
        probe.remove();
    }

    document.addEventListener('input', function (e) {
        var el = e.target;
        if (el.matches('[data-score-range]')) {
            syncPicker(el.closest('[data-score]'), el.value, 'range');
        } else if (el.matches('[data-score-input]')) {
            syncPicker(el.closest('[data-score]'), el.value, 'input');
        } else if (el.matches('[data-counter]')) {
            updateCounter(el);
        }
    });

    // Tidy the typed number (e.g. "7" -> "7.0", "12" -> "10.0") once the user leaves the field.
    document.addEventListener('change', function (e) {
        if (e.target.matches('[data-score-input]')) {
            syncPicker(e.target.closest('[data-score]'), e.target.value || '5', 'change');
        }
    });

    // --- Character counter ---

    function updateCounter(el) {
        var out = document.getElementById(el.getAttribute('data-counter'));
        if (out) out.textContent = el.value.length + ' / ' + el.maxLength;
    }

    // --- Password show/hide ---

    document.addEventListener('click', function (e) {
        var btn = e.target.closest('[data-toggle-password]');
        if (!btn) return;
        var input = document.getElementById(btn.getAttribute('data-toggle-password'));
        if (!input) return;
        var show = input.type === 'password';
        input.type = show ? 'text' : 'password';
        btn.textContent = show ? btn.getAttribute('data-hide-text') : btn.getAttribute('data-show-text');
    });

    // --- Confirmation for destructive actions ---
    // Runs in the capture phase, i.e. before htmx's boosted-form handler on the
    // form itself, so cancelling really stops the request.

    document.addEventListener('submit', function (e) {
        var form = e.target;
        var message = (e.submitter && e.submitter.getAttribute('data-confirm')) || form.getAttribute('data-confirm');
        if (message && !window.confirm(message)) {
            e.preventDefault();
            e.stopImmediatePropagation();
        }
    }, true);

    // --- Show error pages from boosted requests ---
    // htmx ignores 4xx/5xx responses by default, which would silently swallow
    // e.g. "wrong password" or "invalid score" pages. The server always
    // renders a full page for those, so swap them in like any other page.

    document.addEventListener('htmx:beforeSwap', function (e) {
        var status = e.detail.xhr.status;
        if (status >= 400 && status < 600 && e.detail.xhr.responseText) {
            e.detail.shouldSwap = true;
            e.detail.isError = false;
        }
    });

    // --- Page init (first load + every boosted swap) ---

    function init() {
        document.querySelectorAll('[data-score]').forEach(function (root) {
            var input = root.querySelector('[data-score-input]');
            if (input) syncPicker(root, input.value, 'init');
        });
        document.querySelectorAll('[data-counter]').forEach(updateCounter);
        document.querySelectorAll('[data-autohide]').forEach(function (el) {
            setTimeout(function () {
                el.classList.add('is-hiding');
                setTimeout(function () { el.remove(); }, 320);
            }, 3200);
        });
    }

    if (document.fonts && document.fonts.ready) {
        document.fonts.ready.then(function () {
            document.querySelectorAll('[data-score-input]').forEach(fitWidth);
        });
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
    document.addEventListener('htmx:afterSettle', function (e) {
        if (e.target === document.body) init();
    });
})();
