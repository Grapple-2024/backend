/* eslint-disable */
import { approveRequest, denyRequest } from "@/api-requests/request";
import { useMessagingContext } from "@/context/message";
import { useToken } from "@/hook/user";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { useMemo } from "react";
import { Table as BSTable, Button, Spinner } from "react-bootstrap";


interface User {
  id: string;
  first_name: string;
  last_name: string;
  requestor_email: string;
  gymId: string;
  approved?: boolean;
}

interface TableProps {
  data: User[];
};  

const columnHelper = createColumnHelper<User>()

const columns = [
  columnHelper.accessor('requestor_email', {
    header: () => <span>Email</span>,
    maxSize: 20,
    cell: info => {
      return (
        <span>{info.getValue()}</span>
      )
    },
  }),
  columnHelper.accessor('approved', {
    header: () => <></>,
    cell: info => {
      const token = useToken();
      const queryClient = useQueryClient();
      const {
        show,
        setShow,
        setColor,
        setMessage,
      } = useMessagingContext();

      const approve = useMutation({
        mutationKey: ['requests'],
        mutationFn: (id: string) => approveRequest(id, token),
        onSuccess: () => {
          setMessage('Request approved');
          setColor('success');
          setShow(true);

          setTimeout(() => {
            queryClient.invalidateQueries({ queryKey: ['requests'] });
          }, 2000);
        },
        onError: (error: any) => {
          setMessage(error.message);
          setColor('danger');
          setShow(true);
        } 
      });

      const deny = useMutation({
        mutationKey: ['requests'],
        mutationFn: (id: string) => denyRequest(id, token),
        onSuccess: () => {
          setMessage('Request Denied');
          setColor('success');
          setShow(true);

          setTimeout(() => {
            queryClient.invalidateQueries({ queryKey: ['requests'] });
          }, 2000);
        },
        onError: (error: any) => {
          setMessage(error.message);
          setColor('danger');
          setShow(true);
        } 
      });
      
      return (
        <>
          <Button 
            variant='dark' 
            size='sm'
            onClick={() => {
              approve.mutate(info.row.original.id);
            }}
          >
            {show ? <Spinner /> : "Approve"}
          </Button>
          <Button style={{ 
              marginLeft: 10 ,
              backgroundColor: 'white',
              color: 'black'
            }} 
            variant='dark' 
            size='sm'
            onClick={() => {
              deny.mutate(info.row.original.id);
            }}
          >
            {show ? <Spinner /> : "Deny"}
          </Button>
        </>
      );
    },
  }),
];

const MobileTable = ({ 
  defaultData, 
}: any) => {
  const data = useMemo(() => defaultData, [defaultData]);
  
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });
  
  return (
    <div className="p-2" style={{ backgroundColor: '#F1F5F9' }}>
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
                <td key={cell.id} style={{ maxWidth: 100, overflowX: 'auto', whiteSpace: 'nowrap' }}>
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

export default MobileTable;