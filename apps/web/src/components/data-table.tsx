import { useState } from "react"
import {
	flexRender,
	getCoreRowModel,
	getFilteredRowModel,
	getPaginationRowModel,
	useReactTable,
	type ColumnDef,
	type ColumnFiltersState,
	type Row,
} from "@tanstack/react-table"
import { Table, TableHeader, TableBody, TableHead, TableRow, TableCell } from "#/components/ui/table"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent } from "#/components/ui/empty"
import { ChevronLeft, ChevronRight, Search, SearchX } from "lucide-react"

interface DataTableProps<TData> {
	columns: ColumnDef<TData, unknown>[]
	data: TData[]
	filterColumn?: string
	filterPlaceholder?: string
	onRowClick?: (row: Row<TData>) => void
	emptyIcon?: React.ReactNode
	emptyTitle?: string
	emptyDescription?: string
	emptyAction?: React.ReactNode
	pageSize?: number
}

export function DataTable<TData>({
	columns,
	data,
	filterColumn,
	filterPlaceholder = "Search...",
	onRowClick,
	emptyIcon,
	emptyTitle = "No results",
	emptyDescription = "No items match your search.",
	emptyAction,
	pageSize = 20,
}: DataTableProps<TData>) {
	const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])

	const table = useReactTable({
		data,
		columns,
		getCoreRowModel: getCoreRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		onColumnFiltersChange: setColumnFilters,
		state: { columnFilters },
		initialState: { pagination: { pageSize } },
	})

	const filterValue = filterColumn
		? (table.getColumn(filterColumn)?.getFilterValue() as string) ?? ""
		: ""

	const rows = table.getRowModel().rows
	const pageCount = table.getPageCount()
	const pageIndex = table.getState().pagination.pageIndex

	return (
		<div className="space-y-3">
			{filterColumn && (
				<div className="flex items-center gap-2">
					<div className="relative max-w-xs flex-1">
						<Search className="absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
						<Input
							placeholder={filterPlaceholder}
							value={filterValue}
							onChange={(e) => table.getColumn(filterColumn)?.setFilterValue(e.target.value)}
							className="pl-8"
							inputSize="sm"
						/>
					</div>
				</div>
			)}

			{rows.length === 0 ? (
				data.length === 0 ? (
					<Empty className="min-h-60">
						<EmptyHeader>
							{emptyIcon && <EmptyMedia variant="icon">{emptyIcon}</EmptyMedia>}
							<EmptyContent>
								<EmptyTitle>{emptyTitle}</EmptyTitle>
								<EmptyDescription>{emptyDescription}</EmptyDescription>
							</EmptyContent>
						</EmptyHeader>
						{emptyAction}
					</Empty>
				) : (
					<Empty className="min-h-40">
						<EmptyHeader>
							<EmptyMedia variant="icon"><SearchX /></EmptyMedia>
							<EmptyContent>
								<EmptyTitle>No results found</EmptyTitle>
								<EmptyDescription>Try adjusting your search or filter.</EmptyDescription>
							</EmptyContent>
						</EmptyHeader>
					</Empty>
				)
			) : (
				<Table>
					<TableHeader>
						{table.getHeaderGroups().map((headerGroup) => (
							<TableRow key={headerGroup.id}>
								{headerGroup.headers.map((header) => (
									<TableHead key={header.id}>
										{header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
									</TableHead>
								))}
							</TableRow>
						))}
					</TableHeader>
					<TableBody>
						{rows.map((row) => (
							<TableRow
								key={row.id}
								data-state={row.getIsSelected() ? "selected" : undefined}
								className={onRowClick ? "cursor-pointer" : ""}
								onClick={() => onRowClick?.(row)}
							>
								{row.getVisibleCells().map((cell) => (
									<TableCell key={cell.id}>
										{flexRender(cell.column.columnDef.cell, cell.getContext())}
									</TableCell>
								))}
							</TableRow>
						))}
					</TableBody>
				</Table>
			)}

			{pageCount > 1 && (
				<div className="flex items-center justify-between">
					<p className="text-xs text-muted-foreground">
						Page {pageIndex + 1} of {pageCount} &middot; {table.getFilteredRowModel().rows.length} items
					</p>
					<div className="flex items-center gap-1">
						<Button
							variant="outline"
							size="icon-sm"
							onClick={() => table.previousPage()}
							disabled={!table.getCanPreviousPage()}
						>
							<ChevronLeft />
						</Button>
						<Button
							variant="outline"
							size="icon-sm"
							onClick={() => table.nextPage()}
							disabled={!table.getCanNextPage()}
						>
							<ChevronRight />
						</Button>
					</div>
				</div>
			)}
		</div>
	)
}
