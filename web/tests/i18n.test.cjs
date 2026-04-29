/* eslint-disable @typescript-eslint/no-require-imports */
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const ts = require('typescript');

const webRoot = path.join(__dirname, '..');
const srcRoot = path.join(webRoot, 'src');
const localeDir = path.join(webRoot, 'public', 'locale');
const localeFiles = ['zh_hans.json', 'zh_hant.json', 'en.json'];

function readJson(filePath) {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function flattenKeys(value, prefix = '') {
    return Object.entries(value).flatMap(([key, nested]) => {
        const nextKey = prefix ? `${prefix}.${key}` : key;
        if (nested && typeof nested === 'object' && !Array.isArray(nested)) {
            return flattenKeys(nested, nextKey);
        }
        return [nextKey];
    });
}

function collectSourceFiles(dir) {
    const files = [];

    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory()) {
            if (entry.name === 'node_modules') {
                continue;
            }
            files.push(...collectSourceFiles(fullPath));
            continue;
        }

        if (/\.(ts|tsx)$/.test(entry.name) && !entry.name.endsWith('.d.ts')) {
            files.push(fullPath);
        }
    }

    return files;
}

function isLikelyUiText(text) {
    const trimmed = text.trim();
    if (!trimmed) {
        return false;
    }

    if (!/[A-Za-z\p{Script=Han}]/u.test(trimmed)) {
        return false;
    }

    if (/^(use client|button|submit|multiple|single|default|list|advanced|outline|ghost|sm|md|lg|xl|true|false|number|text|url)$/i.test(trimmed)) {
        return false;
    }

    if (/^(https?:\/\/|\/api\/|[A-Z]:\\)/.test(trimmed)) {
        return false;
    }

    if (/^&[a-z]+;$/i.test(trimmed)) {
        return false;
    }

    if (/^[A-Za-z]{1,4}\s*·$/.test(trimmed)) {
        return false;
    }

    if (/^[\[{].*[}\]]$/.test(trimmed)) {
        return false;
    }

    if (/^[A-Za-z0-9_./:-]+$/.test(trimmed) && !trimmed.includes(' ')) {
        return false;
    }

    return true;
}

function findHardcodedUiCopy() {
    const suspects = [];
    const files = collectSourceFiles(srcRoot);

    for (const filePath of files) {
        const source = fs.readFileSync(filePath, 'utf8');
        const sourceFile = ts.createSourceFile(
            filePath,
            source,
            ts.ScriptTarget.Latest,
            true,
            filePath.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS
        );

        const getLine = (node) => sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile)).line + 1;

        const visit = (node) => {
            if (ts.isJsxText(node)) {
                const text = node.getText(sourceFile).trim();
                if (isLikelyUiText(text)) {
                    suspects.push({
                        file: path.relative(webRoot, filePath).replace(/\\/g, '/'),
                        line: getLine(node),
                        kind: 'jsx-text',
                        text,
                    });
                }
            }

            if (ts.isJsxAttribute(node) && ['title', 'aria-label', 'placeholder'].includes(node.name.text)) {
                const initializer = node.initializer;
                const literal =
                    initializer && ts.isStringLiteral(initializer)
                        ? initializer
                        : initializer && ts.isJsxExpression(initializer) && initializer.expression && ts.isStringLiteral(initializer.expression)
                            ? initializer.expression
                            : null;

                if (literal && isLikelyUiText(literal.text)) {
                    suspects.push({
                        file: path.relative(webRoot, filePath).replace(/\\/g, '/'),
                        line: getLine(literal),
                        kind: `attr:${node.name.text}`,
                        text: literal.text,
                    });
                }
            }

            if (ts.isPropertyAssignment(node)) {
                const name = ts.isIdentifier(node.name)
                    ? node.name.text
                    : ts.isStringLiteral(node.name)
                        ? node.name.text
                        : null;

                if (name && ['label', 'title', 'description', 'placeholder', 'hint'].includes(name)) {
                    const initializer = node.initializer;
                    if ((ts.isStringLiteral(initializer) || ts.isNoSubstitutionTemplateLiteral(initializer)) && isLikelyUiText(initializer.text)) {
                        suspects.push({
                            file: path.relative(webRoot, filePath).replace(/\\/g, '/'),
                            line: getLine(initializer),
                            kind: `prop:${name}`,
                            text: initializer.text,
                        });
                    }
                }
            }

            ts.forEachChild(node, visit);
        };

        visit(sourceFile);
    }

    return suspects;
}

test('locale files stay aligned', () => {
    const localeKeySets = localeFiles.map((fileName) => ({
        fileName,
        keys: flattenKeys(readJson(path.join(localeDir, fileName))).sort(),
    }));

    const [baseline, ...rest] = localeKeySets;
    for (const locale of rest) {
        assert.deepStrictEqual(
            locale.keys,
            baseline.keys,
            `${locale.fileName} does not match ${baseline.fileName}`,
        );
    }
});

test('frontend visible copy stays behind i18n', () => {
    const suspects = findHardcodedUiCopy();
    assert.deepStrictEqual(suspects, []);
});
