"use client";

import { useState } from "react";
import { Card } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ContactForm } from "@/components/contact-form";
import { ContactsTable } from "@/components/contacts-table";
import { DeletedContactsTable } from "@/components/deleted-contacts-table";
import { DuplicatesTable } from "@/components/duplicates-table";

export default function Home() {
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const handleContactCreated = () => {
    setRefreshTrigger((prev) => prev + 1);
  };

  return (
    <main className="min-h-screen bg-background py-8 px-4">
      <div className="max-w-6xl mx-auto">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-foreground">
              Contact Manager
            </h1>
            <p className="text-muted-foreground mt-1">
              Manage your contacts efficiently
            </p>
          </div>
          <ContactForm onContactCreated={handleContactCreated} />
        </div>

        <Card className="p-6">
          <Tabs defaultValue="active" className="w-full">
            <TabsList>
              <TabsTrigger value="active">Active Contacts</TabsTrigger>
              {/*<TabsTrigger value="duplicates">Duplicates</TabsTrigger>*/}
              <TabsTrigger value="deleted">Deleted Contacts</TabsTrigger>
            </TabsList>

            <TabsContent value="active" className="mt-6">
              <ContactsTable refreshTrigger={refreshTrigger} />
            </TabsContent>

            {/*<TabsContent value="duplicates" className="mt-6">
              <DuplicatesTable refreshTrigger={refreshTrigger} />
            </TabsContent>*/}

            <TabsContent value="deleted" className="mt-6">
              <DeletedContactsTable refreshTrigger={refreshTrigger} />
            </TabsContent>
          </Tabs>
        </Card>
      </div>
    </main>
  );
}
