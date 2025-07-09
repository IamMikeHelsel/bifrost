import * as vscode from 'vscode';
export interface VirtualTreeItem {
    id: string;
    label: string;
    collapsibleState: vscode.TreeItemCollapsibleState;
    contextValue?: string;
    iconPath?: vscode.ThemeIcon;
    description?: string;
    tooltip?: string;
}
export interface DataProvider<T> {
    getRootItems(): Promise<T[]>;
    getChildren(item: T): Promise<T[]>;
    getTreeItem(item: T): VirtualTreeItem;
}
/**
 * High-performance virtual tree provider that only renders visible items
 */
export declare class VirtualTreeProvider<T> implements vscode.TreeDataProvider<T> {
    protected dataProvider: DataProvider<T>;
    private _onDidChangeTreeData;
    readonly onDidChangeTreeData: vscode.Event<T | undefined | null | void>;
    private cache;
    protected readonly pageSize: number;
    private expandedNodes;
    constructor(dataProvider: DataProvider<T>);
    getTreeItem(element: T): vscode.TreeItem;
    getChildren(element?: T): Promise<T[]>;
    /**
     * Efficient refresh that only updates changed items
     */
    refresh(element?: T): void;
    /**
     * Track node expansion for lazy loading
     */
    onDidExpandElement(element: T): void;
    /**
     * Track node collapse to free memory
     */
    onDidCollapseElement(element: T): void;
    /**
     * Bulk update multiple items efficiently
     */
    bulkUpdate(items: T[]): void;
    /**
     * Search through tree items efficiently
     */
    search(query: string, maxResults?: number): Promise<T[]>;
    /**
     * Get memory usage statistics
     */
    getMemoryStats(): {
        cacheSize: number;
        expandedNodes: number;
    };
    dispose(): void;
}
/**
 * Paginated tree provider for very large datasets
 */
export declare class PaginatedTreeProvider<T> extends VirtualTreeProvider<T> {
    protected readonly paginatedPageSize: number | undefined;
    private pageCache;
    constructor(dataProvider: DataProvider<T>, paginatedPageSize?: number | undefined);
    getChildren(element?: T): Promise<T[]>;
    private getPaginatedChildren;
    loadMoreItems(parentKey: string): Promise<void>;
}
