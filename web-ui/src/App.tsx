import React, { Component } from 'react';
import { BrowserRouter, Routes, Route, Link, useLocation } from 'react-router-dom';
import { Result, Button } from 'antd';
import { TaskCreatePage } from './pages/TaskCreatePage';
import { TaskDetailPage } from './pages/TaskDetailPage';
import { TaskListPage } from './pages/TaskListPage';
import { Layout, Menu } from 'antd';

const { Header, Content, Footer } = Layout;

const NotFound: React.FC = () => (
  <Result
    status="404"
    title="404"
    subTitle="Page not found"
    extra={<Link to="/"><Button type="primary">Back Home</Button></Link>}
  />
);

class ErrorBoundary extends Component<{ children: React.ReactNode }, { hasError: boolean }> {
  constructor(props: { children: React.ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }
  static getDerivedStateFromError() { return { hasError: true }; }
  render() {
    if (this.state.hasError) {
      return <Result status="error" title="Something went wrong" subTitle="Please refresh the page" />;
    }
    return this.props.children;
  }
}

const AppMenu: React.FC = () => {
  const location = useLocation();
  const selectedKey = location.pathname.startsWith('/tasks/new') ? 'create' : 'tasks';
  return (
    <Menu theme="dark" mode="horizontal" selectedKeys={[selectedKey]}>
      <Menu.Item key="tasks"><Link to="/tasks">Tasks</Link></Menu.Item>
      <Menu.Item key="create"><Link to="/tasks/new">Create Task</Link></Menu.Item>
    </Menu>
  );
};

const App: React.FC = () => {
  return (
    <BrowserRouter>
      <Layout className="layout" style={{ minHeight: '100vh' }}>
        <Header>
          <div style={{ float: 'left', color: 'white', fontSize: 20, marginRight: 40 }}>
            DPS - DNS Packets Sender
          </div>
          <AppMenu />
        </Header>
        <Content style={{ padding: 24 }}>
          <ErrorBoundary>
            <Routes>
              <Route path="/" element={<TaskListPage />} />
              <Route path="/tasks" element={<TaskListPage />} />
              <Route path="/tasks/new" element={<TaskCreatePage />} />
              <Route path="/tasks/:id" element={<TaskDetailPage />} />
              <Route path="*" element={<NotFound />} />
            </Routes>
          </ErrorBoundary>
        </Content>
        <Footer style={{ textAlign: 'center', color: '#999' }}>
          Created by Tuzkibug
        </Footer>
      </Layout>
    </BrowserRouter>
  );
};

export default App;