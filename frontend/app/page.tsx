"use client";

import { useState } from "react";
import { Users, Trash2 } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ContactForm } from "@/components/contact-form";
import { ContactsTable } from "@/components/contacts-table";
import { DeletedContactsTable } from "@/components/deleted-contacts-table";
import { Navbar } from "@/components/navbar";

export default function Home() {
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const handleContactCreated = () => {
    setRefreshTrigger((prev) => prev + 1);
  };

  return (
    <div className="min-h-screen bg-background">
      <Navbar />

      <main className="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-8 page-enter">
        {/* ── Page header
 ───────────────────────────────────────── */}
        <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
          <div className="space-y-1">
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Contacts
            </h1>
            <p className="text-sm text-muted-foreground">
              Manage and organise all your contacts in one place.
            </p>
          </div>

          <div className="flex items-center gap-3">
            <ContactForm onContactCreated={handleContactCreated} />
          </div>
        </div>

        {/* ── Tabs ──────────────────────────────────────────────── */}
        <Tabs defaultValue="active" className="w-full">
          {/* Tab bar */}
          <div className="flex items-center justify-between gap-4 border-b border-border pb-0">
            <TabsList className="h-auto bg-transparent p-0 gap-0 rounded-none">
              <TabsTrigger
                value="active"
                className="
                  relative h-11 px-4 rounded-none border-b-2 border-transparent
                  text-sm font-medium text-muted-foreground
                  data-[state=active]:border-primary data-[state=active]:text-primary
                  data-[state=active]:bg-transparent data-[state=active]:shadow-none
                  hover:text-foreground transition-colors duration-150
                  flex items-center gap-2
                "
              >
                <Users size={15} strokeWidth={2} />
                Active Contacts
              </TabsTrigger>

              <TabsTrigger
                value="deleted"
                className="
                  relative h-11 px-4 rounded-none border-b-2 border-transparent
                  text-sm font-medium text-muted-foreground
                  data-[state=active]:border-primary data-[state=active]:text-primary
                  data-[state=active]:bg-transparent data-[state=active]:shadow-none
                  hover:text-foreground transition-colors duration-150
                  flex items-center gap-2
                "
              >
                <Trash2 size={15} strokeWidth={2} />
                Deleted
              </TabsTrigger>
            </TabsList>
          </div>

          {/* Tab content */}
          <TabsContent value="active" className="mt-6 outline-none">
            <ContactsTable refreshTrigger={refreshTrigger} />
          </TabsContent>

          <TabsContent value="deleted" className="mt-6 outline-none">
            <DeletedContactsTable refreshTrigger={refreshTrigger} />
          </TabsContent>
        </Tabs>
      </main>
    </div>
  );
}
