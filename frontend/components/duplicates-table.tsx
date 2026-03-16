'use client'

import { useState, useEffect, useRef } from 'react'
import { getDuplicateContacts, deleteContact } from '@/lib/api-service'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { Empty } from '@/components/ui/empty'
import { toast } from 'sonner'
import { Spinner } from '@/components/ui/spinner'

interface Contact {
  id: string
  name: string
  phone: string
  email: string
  category_id: string
}

interface DuplicatesTableProps {
  refreshTrigger?: number
}

export function DuplicatesTable({ refreshTrigger }: DuplicatesTableProps) {
  const [duplicates, setDuplicates] = useState<Contact[]>([])
  const [loading, setLoading] = useState(true)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const isMounted = useRef(true)

  useEffect(() => {
    isMounted.current = true
    return () => {
      isMounted.current = false
    }
  }, [])

  const fetchDuplicates = async () => {
    try {
      setLoading(true)
      const data = await getDuplicateContacts()
      if (isMounted.current) {
        setDuplicates(data)
      }
    } catch (error) {
      console.error('Error fetching duplicates:', error)
      if (isMounted.current) {
        toast.error('Failed to fetch duplicate contacts')
      }
    } finally {
      if (isMounted.current) {
        setLoading(false)
      }
    }
  }

  useEffect(() => {
    fetchDuplicates()
  }, [refreshTrigger])

  const handleDelete = async (id: string) => {
    try {
      setDeletingId(id)
      await deleteContact(id)
      if (isMounted.current) {
        toast.success('Contact deleted successfully')
        await fetchDuplicates()
      }
    } catch (error) {
      console.error('Error deleting contact:', error)
      if (isMounted.current) {
        toast.error(error instanceof Error ? error.message : 'Failed to delete contact')
      }
    } finally {
      if (isMounted.current) {
        setDeletingId(null)
      }
    }
  }

  // Group duplicates by phone number
  const groupedByPhone = (contacts: Contact[]) => {
    const groups: { [key: string]: Contact[] } = {}
    contacts.forEach((contact) => {
      if (!groups[contact.phone]) {
        groups[contact.phone] = []
      }
      groups[contact.phone].push(contact)
    })
    return Object.keys(groups)
      .sort()
      .reduce((acc, key) => {
        acc[key] = groups[key]
        return acc
      }, {} as { [key: string]: Contact[] })
  }

  if (loading) {
    return (
      <div className="space-y-4">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="space-y-2">
            <Skeleton className="h-6 w-40" />
            <div className="border rounded-lg overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Category</TableHead>
                    <TableHead className="w-20">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {[...Array(2)].map((_, j) => (
                    <TableRow key={j}>
                      <TableCell>
                        <Skeleton className="h-4 w-24" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-4 w-32" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-4 w-20" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-8 w-16" />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
        ))}
      </div>
    )
  }

  if (duplicates.length === 0) {
    return (
      <Empty
        title="No duplicate contacts found"
        description="All your contacts have unique phone numbers"
      />
    )
  }

  const groupedDuplicates = groupedByPhone(duplicates)

  return (
    <div className="space-y-6">
      {Object.entries(groupedDuplicates).map(([phoneNumber, contactsWithPhone]) => (
        <div key={phoneNumber}>
          <h3 className="text-lg font-semibold mb-3 text-foreground">
            Phone: {phoneNumber}
          </h3>
          <div className="border rounded-lg overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Category</TableHead>
                  <TableHead className="w-20">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {contactsWithPhone.map((contact) => (
                  <TableRow key={contact.id}>
                    <TableCell>{contact.name}</TableCell>
                    <TableCell>{contact.email}</TableCell>
                    <TableCell>{contact.category_id}</TableCell>
                    <TableCell>
                      <Button
                        variant="destructive"
                        size="sm"
                        disabled={deletingId === contact.id}
                        onClick={() => handleDelete(contact.id)}
                      >
                        {deletingId === contact.id ? (
                          <Spinner className="h-4 w-4" />
                        ) : (
                          'Delete'
                        )}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      ))}
    </div>
  )
}
