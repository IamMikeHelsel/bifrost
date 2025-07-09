
var __createBinding = (this && this.__createBinding) || (Object.create ? ((o, m, k, k2) => {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: () => m[k] };
    }
    Object.defineProperty(o, k2, desc);
}) : ((o, m, k, k2) => {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? ((o, v) => {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : ((o, v) => {
    o["default"] = v;
}));
var __importStar = (this && this.__importStar) || (() => {
    var ownKeys = (o) => {
        ownKeys = Object.getOwnPropertyNames || ((o) => {
            var ar = [];
            for (var k in o) if (Object.hasOwn(o, k)) ar[ar.length] = k;
            return ar;
        });
        return ownKeys(o);
    };
    return (mod) => {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.PaginatedTreeProvider = exports.VirtualTreeProvider = void 0;
const vscode = __importStar(require("vscode"));
/**
 * High-performance virtual tree provider that only renders visible items
 */
class VirtualTreeProvider {
    dataProvider;
    _onDidChangeTreeData = new vscode.EventEmitter();
    onDidChangeTreeData = this._onDidChangeTreeData.event;
    cache = new Map();
    pageSize = 100;
    expandedNodes = new Set();
    constructor(dataProvider) {
        this.dataProvider = dataProvider;
    }
    getTreeItem(element) {
        const item = this.dataProvider.getTreeItem(element);
        const treeItem = new vscode.TreeItem(item.label, item.collapsibleState);
        treeItem.id = item.id;
        treeItem.contextValue = item.contextValue;
        treeItem.iconPath = item.iconPath;
        treeItem.description = item.description;
        treeItem.tooltip = item.tooltip;
        return treeItem;
    }
    async getChildren(element) {
        if (!element) {
            // Root level
            const cacheKey = '__root__';
            if (this.cache.has(cacheKey)) {
                return this.cache.get(cacheKey);
            }
            const items = await this.dataProvider.getRootItems();
            this.cache.set(cacheKey, items);
            return items;
        }
        const item = this.dataProvider.getTreeItem(element);
        const cacheKey = item.id;
        // Only load children for expanded nodes or when explicitly requested
        if (!this.expandedNodes.has(cacheKey) &&
            item.collapsibleState === vscode.TreeItemCollapsibleState.Collapsed) {
            return [];
        }
        if (this.cache.has(cacheKey)) {
            return this.cache.get(cacheKey);
        }
        const children = await this.dataProvider.getChildren(element);
        this.cache.set(cacheKey, children);
        return children;
    }
    /**
     * Efficient refresh that only updates changed items
     */
    refresh(element) {
        if (element) {
            const item = this.dataProvider.getTreeItem(element);
            this.cache.delete(item.id);
        }
        else {
            this.cache.clear();
        }
        this._onDidChangeTreeData.fire(element);
    }
    /**
     * Track node expansion for lazy loading
     */
    onDidExpandElement(element) {
        const item = this.dataProvider.getTreeItem(element);
        this.expandedNodes.add(item.id);
    }
    /**
     * Track node collapse to free memory
     */
    onDidCollapseElement(element) {
        const item = this.dataProvider.getTreeItem(element);
        this.expandedNodes.delete(item.id);
        this.cache.delete(item.id); // Free memory for collapsed nodes
    }
    /**
     * Bulk update multiple items efficiently
     */
    bulkUpdate(items) {
        // Batch cache invalidation
        items.forEach(item => {
            const treeItem = this.dataProvider.getTreeItem(item);
            this.cache.delete(treeItem.id);
        });
        // Single tree refresh
        this._onDidChangeTreeData.fire();
    }
    /**
     * Search through tree items efficiently
     */
    async search(query, maxResults = 50) {
        const results = [];
        const queryLower = query.toLowerCase();
        const searchInItems = async (items) => {
            for (const item of items) {
                if (results.length >= maxResults)
                    break;
                const treeItem = this.dataProvider.getTreeItem(item);
                if (treeItem.label.toLowerCase().includes(queryLower)) {
                    results.push(item);
                }
                // Search children if expanded
                if (this.expandedNodes.has(treeItem.id)) {
                    const children = await this.dataProvider.getChildren(item);
                    await searchInItems(children);
                }
            }
        };
        const rootItems = await this.dataProvider.getRootItems();
        await searchInItems(rootItems);
        return results;
    }
    /**
     * Get memory usage statistics
     */
    getMemoryStats() {
        return {
            cacheSize: this.cache.size,
            expandedNodes: this.expandedNodes.size
        };
    }
    dispose() {
        this.cache.clear();
        this.expandedNodes.clear();
        this._onDidChangeTreeData.dispose();
    }
}
exports.VirtualTreeProvider = VirtualTreeProvider;
/**
 * Paginated tree provider for very large datasets
 */
class PaginatedTreeProvider extends VirtualTreeProvider {
    paginatedPageSize;
    pageCache = new Map();
    constructor(dataProvider, paginatedPageSize = 100) {
        super(dataProvider);
        this.paginatedPageSize = paginatedPageSize;
    }
    async getChildren(element) {
        if (!element) {
            return this.getPaginatedChildren('__root__', () => this.dataProvider.getRootItems());
        }
        const item = this.dataProvider.getTreeItem(element);
        return this.getPaginatedChildren(item.id, () => this.dataProvider.getChildren(element));
    }
    async getPaginatedChildren(cacheKey, fetchFn) {
        if (this.pageCache.has(cacheKey)) {
            const cached = this.pageCache.get(cacheKey);
            return cached.items;
        }
        const allItems = await fetchFn();
        // For now, return first page
        // TODO: Implement "Load More" functionality
        const pageItems = allItems.slice(0, this.paginatedPageSize);
        this.pageCache.set(cacheKey, {
            items: pageItems,
            totalCount: allItems.length,
            page: 0
        });
        return pageItems;
    }
    async loadMoreItems(parentKey) {
        const cached = this.pageCache.get(parentKey);
        if (!cached)
            return;
        const nextPage = cached.page + 1;
        const startIndex = nextPage * this.paginatedPageSize;
        // TODO: Implement actual paginated loading
        // For now, this is a placeholder
        this.refresh();
    }
}
exports.PaginatedTreeProvider = PaginatedTreeProvider;
//# sourceMappingURL=virtualTreeProvider.js.map