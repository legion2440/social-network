(function (root, factory) {
  'use strict';
  const api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  if (root) root.LoopDesktopSearch = api;
})(typeof window !== 'undefined' ? window : globalThis, function () {
  'use strict';

  function tokenize(query) {
    const source = String(query || '').trim();
    if (!source) return [];
    const tokens = [];
    const pattern = /(?:include:|exclude:|fuzzy:|[+\-~])?"[^"]*"|(?:include:|exclude:|fuzzy:)?'[^']*'|\S+/gi;
    let match;
    while ((match = pattern.exec(source)) !== null) tokens.push(match[0]);
    return tokens;
  }

  function unquote(value) {
    const text = String(value || '').trim();
    if (text.length >= 2) {
      const first = text[0];
      const last = text[text.length - 1];
      if ((first === '"' && last === '"') || (first === "'" && last === "'")) {
        return text.slice(1, -1);
      }
    }
    return text;
  }

  function parseQuery(query) {
    return tokenize(query).map(token => {
      const numeric = token.match(/^(>=|<=|!=|=|>|<)(-?(?:\d+(?:[.,]\d+)?|\.\d+))$/);
      if (numeric) {
        return {
          type: 'number',
          operator: numeric[1],
          value: Number(numeric[2].replace(',', '.'))
        };
      }

      const named = token.match(/^(include|exclude|fuzzy):(.*)$/i);
      if (named) {
        return {
          type: named[1].toLowerCase(),
          value: unquote(named[2]).toLocaleLowerCase()
        };
      }

      const prefix = token[0];
      if (prefix === '+' || prefix === '-' || prefix === '~') {
        return {
          type: prefix === '+' ? 'include' : prefix === '-' ? 'exclude' : 'fuzzy',
          value: unquote(token.slice(1)).toLocaleLowerCase()
        };
      }

      return { type: 'include', value: unquote(token).toLocaleLowerCase() };
    }).filter(filter => filter.type === 'number' || filter.value !== '');
  }

  function wordTokens(text) {
    const value = String(text || '').toLocaleLowerCase();
    try {
      return value.match(/[\p{L}\p{N}_'-]+/gu) || [];
    } catch (_error) {
      return value.match(/[a-zа-яё0-9_'-]+/gi) || [];
    }
  }

  function withinDistance(a, b, limit) {
    if (a === b) return true;
    if (Math.abs(a.length - b.length) > limit) return false;
    let previous = Array.from({ length: b.length + 1 }, (_unused, index) => index);
    for (let i = 1; i <= a.length; i += 1) {
      const current = [i];
      let rowMin = i;
      for (let j = 1; j <= b.length; j += 1) {
        const cost = a[i - 1] === b[j - 1] ? 0 : 1;
        const value = Math.min(
          current[j - 1] + 1,
          previous[j] + 1,
          previous[j - 1] + cost
        );
        current.push(value);
        if (value < rowMin) rowMin = value;
      }
      if (rowMin > limit) return false;
      previous = current;
    }
    return previous[b.length] <= limit;
  }

  function fuzzyMatch(text, needle) {
    const query = String(needle || '').toLocaleLowerCase();
    if (!query) return true;
    const normalized = String(text || '').toLocaleLowerCase();
    if (normalized.includes(query)) return true;
    const limit = Math.min(3, Math.max(1, Math.floor(query.length * 0.3)));
    return wordTokens(normalized).some(word => withinDistance(word, query, limit));
  }

  function extractNumbers(text) {
    const matches = String(text || '').match(/-?(?:\d+(?:[.,]\d+)?|\.\d+)/g) || [];
    return matches.map(value => Number(value.replace(',', '.'))).filter(Number.isFinite);
  }

  function numberEquals(left, right) {
    return Math.abs(left - right) <= Number.EPSILON * Math.max(1, Math.abs(left), Math.abs(right));
  }

  function numberMatches(numbers, operator, expected) {
    if (!numbers.length || !Number.isFinite(expected)) return false;
    switch (operator) {
      case '=': return numbers.some(value => numberEquals(value, expected));
      case '!=': return numbers.every(value => !numberEquals(value, expected));
      case '>': return numbers.some(value => value > expected);
      case '<': return numbers.some(value => value < expected);
      case '>=': return numbers.some(value => value > expected || numberEquals(value, expected));
      case '<=': return numbers.some(value => value < expected || numberEquals(value, expected));
      default: return false;
    }
  }

  function matchesMessage(text, query) {
    const normalized = String(text || '').toLocaleLowerCase();
    const filters = parseQuery(query);
    if (!filters.length) return true;
    let numbers = null;

    return filters.every(filter => {
      if (filter.type === 'include') return normalized.includes(filter.value);
      if (filter.type === 'exclude') return !normalized.includes(filter.value);
      if (filter.type === 'fuzzy') return fuzzyMatch(normalized, filter.value);
      if (filter.type === 'number') {
        if (numbers === null) numbers = extractNumbers(normalized);
        return numberMatches(numbers, filter.operator, filter.value);
      }
      return true;
    });
  }

  return {
    extractNumbers,
    fuzzyMatch,
    matchesMessage,
    parseQuery
  };
});
