(function () {
  'use strict';

  function extractEnumWords(tmGrammar) {
    try {
      var pattern = tmGrammar.repository.enumValues.patterns[0].match;
      var m = pattern.match(/\(\?:([^)]*)\)/);
      if (!m || !m[1]) return [];
      return m[1].split('|').map(function (s) { return s.trim(); }).filter(Boolean);
    } catch (e) {
      return [];
    }
  }

  function toMonarch(enumWords) {
    var enums = enumWords.length ? enumWords : ['left', 'right', 'top', 'bottom'];
    return {
      defaultToken: '',
      tokenPostfix: '.sml',
      keywords: ['Page', 'Row', 'Column', 'Card', 'Link', 'Markdown', 'Image', 'Spacer'],
      typeKeywords: ['true', 'false'],
      enumKeywords: enums,
      tokenizer: {
        root: [
          [/\/[\*]/, { token: 'comment', next: '@comment' }],
          [/\/\/.*/, 'comment'],
          [/@[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)+/, 'variable.predefined'],
          [/"([^"\\]|\\.)*$/, 'string.invalid'],
          [/"/, { token: 'string.quote', next: '@string' }],
          [/\b(?:true|false)\b/, 'keyword'],
          [/\b(?:Page|Row|Column|Card|Link|Markdown|Image|Spacer)\b(?=\s*\{)/, 'type.identifier'],
          [/\b[A-Za-z_][A-Za-z0-9_.-]*\b(?=\s*:)/, 'variable'],
          [/\b(?:-?(?:\d+\.\d+|\.\d+|\d+))\b/, 'number'],
          [new RegExp('\\b(?:' + enums.join('|').replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')\\b'), 'keyword.control'],
          [/[{}]/, 'delimiter.bracket'],
          [/[\[\]()]/, 'delimiter'],
          [/,/, 'delimiter'],
          [/:/, 'delimiter'],
        ],
        comment: [
          [/[^/*]+/, 'comment'],
          [/\*\//, { token: 'comment', next: '@pop' }],
          [/[/*]/, 'comment']
        ],
        string: [
          [/[^\\"]+/, 'string'],
          [/\\["nrt\\]/, 'string.escape'],
          [/\\./, 'string.escape.invalid'],
          [/"/, { token: 'string.quote', next: '@pop' }]
        ]
      }
    };
  }

  function initMonaco() {
    var cfg = window.SML_EDITOR_CONFIG || {};
    var textarea = document.getElementById('sml_source');
    var editorHost = document.getElementById('sml_monaco_editor');
    if (!textarea || !editorHost || typeof require === 'undefined') {
      return;
    }

    require.config({ paths: { vs: cfg.vsPath } });
    require(['vs/editor/editor.main'], function () {
      var languageId = cfg.languageId || 'sml';
      var enumWords = extractEnumWords(cfg.tmGrammar || {});

      monaco.languages.register({ id: languageId });
      monaco.languages.setLanguageConfiguration(languageId, cfg.languageConfiguration || {});
      monaco.languages.setMonarchTokensProvider(languageId, toMonarch(enumWords));

      editorHost.style.height = '460px';
      textarea.style.display = 'none';

      var editor = monaco.editor.create(editorHost, {
        value: textarea.value || '',
        language: languageId,
        theme: 'vs-dark',
        minimap: { enabled: false },
        automaticLayout: true,
        fontSize: 14,
        tabSize: 2,
        insertSpaces: true,
        scrollBeyondLastLine: false,
      });

      var form = textarea.closest('form');
      if (form) {
        form.addEventListener('submit', function () {
          textarea.value = editor.getValue();
        });
      }
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initMonaco);
  } else {
    initMonaco();
  }
})();
