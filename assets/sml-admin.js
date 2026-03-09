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

  function toSmlMonarch(enumWords) {
    var enums = enumWords.length ? enumWords : ['left', 'right', 'top', 'bottom'];
    return {
      defaultToken: '',
      tokenPostfix: '.sml',
      keywords: ['Page', 'Hero', 'Row', 'Column', 'Card', 'Link', 'Markdown', 'Image', 'Spacer', 'Assets', 'Head', 'Foot', 'CssTemplate', 'JsTemplate'],
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
          [/\b(?:Page|Hero|Row|Column|Card|Link|Markdown|Image|Spacer|Assets|Head|Foot|CssTemplate|JsTemplate)\b(?=\s*\{)/, 'type.identifier'],
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

  function toTwigMonarch() {
    return {
      defaultToken: '',
      tokenPostfix: '.twig',
      tokenizer: {
        root: [
          [/\{#/, { token: 'comment.twig', next: '@twigComment' }],
          [/\{\{[-~]?/, { token: 'delimiter.twig', next: '@twigOutput' }],
          [/\{%[-~]?/, { token: 'delimiter.twig', next: '@twigTag' }],
          [/[^\{]+/, ''],
          [/\{/, '']
        ],
        twigComment: [
          [/#\}/, { token: 'comment.twig', next: '@pop' }],
          [/./, 'comment.twig']
        ],
        twigOutput: [
          [/[-~]?\}\}/, { token: 'delimiter.twig', next: '@pop' }],
          [/\|[A-Za-z_][A-Za-z0-9_]*/, 'type.identifier'],
          [/\b(?:true|false|null)\b/, 'keyword'],
          [/"([^"\\]|\\.)*"/, 'string'],
          [/'([^'\\]|\\.)*'/, 'string'],
          [/\b\d+(?:\.\d+)?\b/, 'number'],
          [/\b[A-Za-z_][A-Za-z0-9_]*\b/, 'variable']
        ],
        twigTag: [
          [/[-~]?%\}/, { token: 'delimiter.twig', next: '@pop' }],
          [/\b(?:if|endif|for|endfor|set|block|endblock|extends|include|with|endwith|else|elseif|macro|endmacro)\b/, 'keyword'],
          [/\b(?:true|false|null)\b/, 'keyword'],
          [/"([^"\\]|\\.)*"/, 'string'],
          [/'([^'\\]|\\.)*'/, 'string'],
          [/\b\d+(?:\.\d+)?\b/, 'number'],
          [/\b[A-Za-z_][A-Za-z0-9_]*\b/, 'variable']
        ]
      }
    };
  }

  function createEditor(hostId, textareaId, languageId, options) {
    var textarea = document.getElementById(textareaId);
    var host = document.getElementById(hostId);
    if (!textarea || !host) {
      return null;
    }

    host.style.height = (options && options.height) || '460px';
    textarea.style.display = 'none';

    var editor = monaco.editor.create(host, {
      value: textarea.value || '',
      language: languageId,
      theme: 'vs-dark',
      minimap: { enabled: false },
      automaticLayout: true,
      fontSize: 14,
      tabSize: 2,
      insertSpaces: true,
      scrollBeyondLastLine: false,
      wordWrap: (options && options.wordWrap) || 'off'
    });

    return {
      textarea: textarea,
      editor: editor
    };
  }

  function initMonaco() {
    var cfg = window.SML_EDITOR_CONFIG || {};
    if (typeof require === 'undefined') {
      return;
    }

    require.config({ paths: { vs: cfg.vsPath } });
    require(['vs/editor/editor.main'], function () {
      var smlLanguageId = cfg.languageId || 'sml';
      var enumWords = extractEnumWords(cfg.tmGrammar || {});

      monaco.languages.register({ id: smlLanguageId });
      monaco.languages.setLanguageConfiguration(smlLanguageId, cfg.languageConfiguration || {});
      monaco.languages.setMonarchTokensProvider(smlLanguageId, toSmlMonarch(enumWords));

      var twigLanguageId = 'sml-twig';
      monaco.languages.register({ id: twigLanguageId });
      monaco.languages.setMonarchTokensProvider(twigLanguageId, toTwigMonarch());

      var editors = [];
      var sml = createEditor('sml_monaco_editor', 'sml_source', smlLanguageId, { height: '460px', wordWrap: 'on' });
      if (sml) editors.push(sml);

      var twigTemplate = createEditor('sml_template_monaco_editor', 'sml_template_source', twigLanguageId, { height: '420px', wordWrap: 'on' });
      if (twigTemplate) editors.push(twigTemplate);

      var markdownPart = createEditor('sml_markdown_monaco_editor', 'sml_markdown_source', 'markdown', { height: '420px', wordWrap: 'on' });
      if (markdownPart) editors.push(markdownPart);

      if (!editors.length) {
        return;
      }

      var form = editors[0].textarea.closest('form');
      if (form) {
        form.addEventListener('submit', function () {
          editors.forEach(function (entry) {
            entry.textarea.value = entry.editor.getValue();
          });
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
