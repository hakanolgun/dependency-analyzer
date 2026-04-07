(function () {
  'use strict';

  // ── State ────────────────────────────────────────────────────
  var sortKey = null;
  var sortDir = 'desc'; // 'asc' | 'desc'

  // ── Sort helpers ─────────────────────────────────────────────
  var maintainedWeight = { yes: 3, unlikely: 2, no: 1 };

  function valueOf(dep, key) {
    switch (key) {
      case 'name':            return (dep.name || '').toLowerCase();
      case 'weeklyDownloads': return dep.weeklyDownloads != null ? dep.weeklyDownloads : -1;
      case 'lastUpdateDate':  return dep.lastUpdateDate ? new Date(dep.lastUpdateDate).getTime() : 0;
      case 'isMaintained':    return maintainedWeight[dep.isMaintained] || 0;
      case 'score':           return dep.score != null ? dep.score : -1;
      case 'newArchitecture': return dep.newArchitecture ? 1 : 0;
      default:                return 0;
    }
  }

  function sortedDeps() {
    var deps = REPORT_DATA.dependencies.slice();
    if (!sortKey) return deps;
    deps.sort(function (a, b) {
      var va = valueOf(a, sortKey);
      var vb = valueOf(b, sortKey);
      if (va < vb) return sortDir === 'asc' ? -1 : 1;
      if (va > vb) return sortDir === 'asc' ?  1 : -1;
      return 0;
    });
    return deps;
  }

  // ── Badge helpers ─────────────────────────────────────────────
  function maintainedBadge(status) {
    if (!status) return '<span class="muted">-</span>';
    var cls = status === 'yes' ? 'success' : status === 'unlikely' ? 'warning' : 'danger';
    var txt = status === 'yes' ? 'Yes' : status === 'unlikely' ? 'Unlikely' : 'No';
    return '<span class="badge ' + cls + '">' + txt + '</span>';
  }

  function replaceabilityCell(dep) {
    if (dep.error) return '<span class="muted">-</span>';
    var score = dep.score != null ? dep.score : 0;
    var label, cls;
    if (score >= 71)      { label = 'Hard';   cls = 'danger'; }
    else if (score >= 31) { label = 'Medium'; cls = 'warning'; }
    else                  { label = 'Easy';   cls = 'success'; }

    return '<div class="replace-stack">' +
      '<span class="badge ' + cls + '">' + label + '</span>' +
      '<span class="sub">' + score + '/100</span>' +
      '</div>';
  }

  function newArchCell(dep) {
    if (dep.newArchitecture === true)  return '<span class="na-yes">Yes</span>';
    if (dep.newArchitecture === false) return '<span class="na-no">No</span>';
    return '<span class="muted">-</span>';
  }

  function escapeHTML(s) {
    return String(s)
      .replace(/&/g,'&amp;').replace(/</g,'&lt;')
      .replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  }

  function formatDownloads(n) {
    if (n == null) return '-';
    return Math.floor(n / 1000) + ' K';
  }

  function formatDate(dep) {
    if (!dep.timeSinceLastUpdate) return '<span class="muted">-</span>';
    var secondary = '';
    if (dep.lastUpdateDate) {
      var d = new Date(dep.lastUpdateDate);
      secondary = (d.getMonth()+1) + '/' + d.getDate() + '/' + d.getFullYear();
    }
    return '<div class="stack"><span>' + escapeHTML(dep.timeSinceLastUpdate) + '</span>' +
           (secondary ? '<span class="sub">' + secondary + '</span>' : '') + '</div>';
  }

  // ── Render table body ─────────────────────────────────────────
  var hasRN = REPORT_DATA.hasReactNative;

  function renderBody() {
    var deps = sortedDeps();
    var rows = deps.map(function (dep) {
      var namecell;
      if (dep.repoUrl) {
        namecell = '<a href="' + escapeHTML(dep.repoUrl) + '" target="_blank" rel="noreferrer" class="repo-link" title="' + escapeHTML(dep.repoUrl) + '">' +
                   escapeHTML(dep.name) + ' <span class="ext">↗</span></a>';
      } else {
        namecell = '<span style="word-break:break-all;">' + escapeHTML(dep.name) + '</span>';
      }
      var errCell = dep.error ? '<span class="pkg-err">' + escapeHTML(dep.error) + '</span>' : '';

      var latestCell = dep.latestVersion
        ? '<span class="badge info">' + escapeHTML(dep.latestVersion) + '</span>'
        : '<span class="muted">-</span>';

      var rnCell = hasRN ? '<td class="cell-center">' + newArchCell(dep) + '</td>' : '';

      return '<tr>' +
        '<td><div class="pkg-name">' + namecell + '</div>' + errCell + '</td>' +
        '<td><span class="badge gray">' + escapeHTML(dep.version.replace(/[\^~]/,'')) + '</span></td>' +
        '<td>' + latestCell + '</td>' +
        '<td>' + formatDownloads(dep.weeklyDownloads) + '</td>' +
        '<td>' + formatDate(dep) + '</td>' +
        '<td>' + maintainedBadge(dep.isMaintained) + '</td>' +
        '<td class="cell-replace">' + replaceabilityCell(dep) + '</td>' +
        rnCell +
        '</tr>';
    });
    document.getElementById('results-body').innerHTML = rows.join('');
  }

  // ── Sort-icon update ──────────────────────────────────────────
  function renderSortIcons() {
    document.querySelectorAll('.sort-icon').forEach(function (el) {
      var col = el.getAttribute('data-for');
      if (col === sortKey) {
        el.textContent = sortDir === 'asc' ? ' ↑' : ' ↓';
      } else {
        el.textContent = '';
      }
    });
  }

  // ── Header click handlers ─────────────────────────────────────
  document.querySelectorAll('th[data-col]').forEach(function (th) {
    th.addEventListener('click', function () {
      var col = th.getAttribute('data-col');
      if (sortKey === col) {
        if (sortDir === 'asc') {
          sortDir = 'desc';
        } else {
          // Third state: back to original order
          sortKey = null;
        }
      } else {
        // First state: ASC for all columns (including Name)
        sortKey = col;
        sortDir = 'asc';
      }
      renderSortIcons();
      renderBody();
    });
  });

  // ── Download JSON ─────────────────────────────────────────────
  document.getElementById('btn-download').addEventListener('click', function () {
    var timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    var payload = {
      generatedAt: new Date().toISOString(),
      ecosystem: 'npm',
      sort: { key: sortKey, direction: sortDir },
      results: sortedDeps()
    };
    var blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
    var url = URL.createObjectURL(blob);
    var a = document.createElement('a');
    a.href = url;
    a.download = 'dep-analysis-' + timestamp + '.json';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  });

  // Initial render syncs with the server-rendered HTML (no-op on first load,
  // but ensures body is always driven by JS after page paint).
  renderSortIcons();

})();
