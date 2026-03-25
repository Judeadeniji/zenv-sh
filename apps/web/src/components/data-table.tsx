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
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent } from "#/components/ui/empty"
import { ChevronLeft, ChevronRight, SearchX } from "lucide-react"

interface DataTableProps<TData> {
	columns: ColumnDef<TData, unknown>[]
	data: TData[]
	onRowClick?: (row: Row<TData>) => void
	emptyIcon?: React.ReactNode
	emptyTitle?: string
	emptyDescription?: string
	emptyAction?: React.ReactNode
	pageSize?: number
	/** Server-side pagination — pass these to let the server control paging. */
	pagination?: {
		page: number
		totalPages: number
		total: number
		onPageChange: (page: number) => void
	}
}

export function DataTable<TData>({
	columns,
	data,
	onRowClick,
	emptyIcon,
	emptyTitle = "No results",
	emptyDescription = "No items match your search.",
	emptyAction,
	pageSize = 20,
	pagination,
}: DataTableProps<TData>) {
	const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
	const isServerPaginated = !!pagination

	const table = useReactTable({
		data,
		columns,
		getCoreRowModel: getCoreRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		...(!isServerPaginated && { getPaginationRowModel: getPaginationRowModel() }),
		onColumnFiltersChange: setColumnFilters,
		state: { columnFilters },
		initialState: { pagination: { pageSize } },
		manualPagination: isServerPaginated,
		...(isServerPaginated && { pageCount: pagination.totalPages }),
	})

	const rows = table.getRowModel().rows

	// Pagination state — server or client
	const showPagination = isServerPaginated
		? pagination.totalPages > 1
		: table.getPageCount() > 1
	const currentPage = isServerPaginated ? pagination.page : table.getState().pagination.pageIndex + 1
	const totalPages = isServerPaginated ? pagination.totalPages : table.getPageCount()
	const totalItems = isServerPaginated ? pagination.total : table.getFilteredRowModel().rows.length

	const handlePrev = () => {
		if (isServerPaginated) pagination.onPageChange(pagination.page - 1)
		else table.previousPage()
	}
	const handleNext = () => {
		if (isServerPaginated) pagination.onPageChange(pagination.page + 1)
		else table.nextPage()
	}
	const canPrev = isServerPaginated ? pagination.page > 1 : table.getCanPreviousPage()
	const canNext = isServerPaginated ? pagination.page < pagination.totalPages : table.getCanNextPage()

	return (
		<div className="space-y-3">
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

			{showPagination && (
				<div className="flex items-center justify-between">
					<p className="text-xs text-muted-foreground">
						{totalItems} entries &middot; page {currentPage} of {totalPages}
					</p>
					<div className="flex items-center gap-1">
						<Button variant="outline" size="icon-sm" onClick={handlePrev} disabled={!canPrev}>
							<ChevronLeft />
						</Button>
						<Button variant="outline" size="icon-sm" onClick={handleNext} disabled={!canNext}>
							<ChevronRight />
						</Button>
					</div>
				</div>
			)}
		</div>
	)
}
