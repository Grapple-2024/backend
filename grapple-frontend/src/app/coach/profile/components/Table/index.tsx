import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { useReducer, useState } from "react";
import { Table as BSTable, Button } from "react-bootstrap";


interface User {
  pk: string;
  first_name: string;
  last_name: string;
  email: string;
  gymId: string;
  approved?: boolean;
}

interface TableProps {
  data: User[];
};  

const columnHelper = createColumnHelper<User>()

const columns = [
  columnHelper.accessor('first_name', {
    cell: info => info.getValue(),
    header: () => <span>First</span>,
  }),
  columnHelper.accessor('last_name', {
    header: () => <span>Last</span>,
    cell: info => info.getValue(),
  }),
  columnHelper.accessor('email', {
    header: () => <span>Email</span>,
    cell: info => info.getValue(),
  }),
];

const Table = ({ defaultData }: any) => {
  const [data, setData] = useState(() => defaultData);
  
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });
  
  return (
    <div className="p-2">
      <BSTable striped>
        <thead>
          {table.getHeaderGroups().map(headerGroup => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map(header => (
                <th key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.map(row => (
            <tr key={row.id}>
              {row.getVisibleCells().map(cell => (
                <td key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </BSTable>
    </div>
  );
};

export default Table;