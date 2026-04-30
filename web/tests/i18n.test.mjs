import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const webRoot = path.resolve(__dirname, '..');
const localeDir = path.join(webRoot, 'public', 'locale');
const localeFiles = ['en.json', 'zh_hans.json', 'zh_hant.json'];

function readJson(filePath) {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function collectKeys(value, prefix = '', keys = new Set()) {
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
        return keys;
    }

    for (const [key, child] of Object.entries(value)) {
        const next = prefix ? `${prefix}.${key}` : key;
        keys.add(next);
        collectKeys(child, next, keys);
    }

    return keys;
}

function assertLocaleParity() {
    const [baseName, ...restNames] = localeFiles;
    const base = collectKeys(readJson(path.join(localeDir, baseName)));

    for (const name of restNames) {
        const current = collectKeys(readJson(path.join(localeDir, name)));
        const missing = [...base].filter((key) => !current.has(key));
        const extra = [...current].filter((key) => !base.has(key));
        assert.deepEqual(missing, [], `${name} missing locale keys:\n${missing.join('\n')}`);
        assert.deepEqual(extra, [], `${name} has extra locale keys:\n${extra.join('\n')}`);
    }
}

function assertNoHardcodedCopy(relativePath, forbiddenSnippets) {
    const content = fs.readFileSync(path.join(webRoot, relativePath), 'utf8');
    for (const snippet of forbiddenSnippets) {
        assert.equal(
            content.includes(snippet),
            false,
            `${relativePath} still contains hardcoded copy: ${snippet}`,
        );
    }
}

function run() {
    assertLocaleParity();
    assertNoHardcodedCopy('src/components/modules/group/Editor.tsx', [
        'API 分类',
        'Condition (JSON)',
        'aria-label="search"',
    ]);
    assertNoHardcodedCopy('src/components/modules/channel/Form.tsx', [
        'title="Remove"',
    ]);
    assertNoHardcodedCopy('src/components/modules/channel/templates.ts', [
        "description: '",
    ]);
}

run();
console.log('i18n checks passed');
